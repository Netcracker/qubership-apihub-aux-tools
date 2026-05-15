package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"apihub-portal-package-copy/internal/apihub"
	"apihub-portal-package-copy/internal/copy"
	"apihub-portal-package-copy/internal/logx"
	"apihub-portal-package-copy/internal/scope"
	"apihub-portal-package-copy/internal/state"
	"apihub-portal-package-copy/internal/versionid"
)

func main() {
	srcURL := flag.String("source-url", "", "Source APIHUB base URL")
	srcKey := flag.String("source-api-key", "", "Source Apihub API key (api-key header)")
	srcWS := flag.String("source-workspace-id", "", "Source workspace package id")
	srcRoot := flag.String("source-root-id", "", "Source package or group id, or * for entire workspace — may be a suffix under --source-workspace-id (canonical id is inferred)")
	versionsStr := flag.String("versions", "", "Comma-separated version names, or * for all published versions per package")
	excludeStr := flag.String("exclude-packages", "", "Optional comma-separated source package or group ids to skip — same suffix resolution as --source-root-id under --source-workspace-id")

	tgtURL := flag.String("target-url", "", "Target APIHUB base URL")
	tgtKey := flag.String("target-api-key", "", "Target Apihub API key")
	tgtWS := flag.String("target-workspace-id", "", "Target workspace package id")

	workDir := flag.String("work-dir", "", "Working directory for snapshot + manifest (required)")
	retryAfterFail := flag.Bool("retry-after-fail", false, "Skip re-fetch; continue publish from manifest checkpoint")
	forceRefresh := flag.Bool("force-refresh-fetch", false, "Re-download sources/config even if already fetched")
	insecureTLS := flag.Bool("insecure-skip-tls-verify", false, "Do not verify HTTPS server certificates (insecure; dev / private CA only)")

	noColor := flag.Bool("no-color", false, "Disable colored log output")
	debug := flag.Bool("debug", false, "Verbose HTTP trace and detailed version-match diagnostics (stderr-only)")

	flag.Parse()

	if *noColor {
		logx.DisableColor()
	}
	if *debug {
		logx.SetDebug(true)
		apihub.VerboseHTTP = true
		apihub.HTTPDebug = logx.Debugf
	}

	if *workDir == "" || *srcURL == "" || *srcKey == "" || *srcWS == "" || *srcRoot == "" ||
		*versionsStr == "" || *tgtURL == "" || *tgtKey == "" || *tgtWS == "" {
		flag.Usage()
		os.Exit(2)
	}
	vers, err := scope.NormalizeVersionTokens(splitComma(*versionsStr))
	if err != nil {
		logx.Fatal(err)
	}
	if !scope.IsAllVersions(vers) {
		vers = scope.DedupSorted(vers)
	}
	srcWSCanon := strings.TrimSpace(*srcWS)
	srcRootCanon := scope.QualifyUnderWorkspace(srcWSCanon, *srcRoot)
	if srcRootCanon != strings.TrimSpace(*srcRoot) {
		logx.Notef("Resolved --source-root-id under workspace: %q → %q", *srcRoot, srcRootCanon)
	}
	excluded := scope.QualifyExcludedUnderWorkspace(srcWSCanon, scope.DedupSorted(splitComma(*excludeStr)))

	if err := os.MkdirAll(filepath.Join(*workDir, "data"), 0755); err != nil {
		logx.Fatal(err)
	}

	srcCl := apihub.New(*srcURL, *srcKey, *insecureTLS)
	tgtCl := apihub.New(*tgtURL, *tgtKey, *insecureTLS)

	m, err := loadOrInitManifest(*workDir, *srcWS, *tgtWS, srcRootCanon, vers, excluded, *retryAfterFail, *forceRefresh)
	if err != nil {
		logx.Fatal(err)
	}

	logx.Section("APIHUB portal package copy")
	logx.Infof("Work dir: %s", *workDir)

	if !m.FetchComplete || *forceRefresh {
		logx.Section("Fetch")
		logx.Step("Sources + publish config from source")
		if err := runFetch(srcCl, m, *srcWS, srcRootCanon, vers, excluded, *workDir, *forceRefresh); err != nil {
			logx.Fatalf("fetch failed: %v", err)
		}
		m.FetchComplete = true
		if err := state.SaveManifest(*workDir, m); err != nil {
			logx.Fatal(err)
		}
		logx.Okf("Fetched snapshot.")
	} else {
		logx.Section("Fetch")
		logx.Infof("Skipping download — snapshot already complete (use --force-refresh-fetch to re-fetch).")
	}

	if err := runPlanLog(m, *workDir); err != nil {
		logx.Fatalf("plan failed: %v", err)
	}

	logx.Section("Apply")
	logx.Step("Create hierarchy on target & publish versions")
	if err := runApply(srcCl, tgtCl, m, *workDir, *srcWS, *tgtWS, *retryAfterFail); err != nil {
		logx.Fatalf("apply failed: %v", err)
	}
	if err := state.SaveManifest(*workDir, m); err != nil {
		logx.Fatal(err)
	}
	logx.Done()
}

func splitComma(s string) []string {
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func dataSubdir(pkg, ver string) string {
	return filepath.Join(fsSafeSegment(pkg), fsSafeSegment(ver))
}

// fsSafeSegment maps a package or version id to a single path segment (readable; illegal FS chars → '_').
// Very long segments are truncated with a short hash suffix to avoid collisions and path limits.
func fsSafeSegment(s string) string {
	const maxSeg = 180
	s = strings.TrimSpace(s)
	if s == "" {
		return "_"
	}
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|', '\x00':
			b.WriteByte('_')
		default:
			b.WriteRune(r)
		}
	}
	out := strings.Trim(b.String(), `. `)
	if out == "" {
		return "_"
	}
	if len(out) > maxSeg {
		sum := sha256.Sum256([]byte(s))
		suf := hex.EncodeToString(sum[:6])
		keep := maxSeg - 1 - len(suf)
		if keep < 32 {
			keep = 32
		}
		out = out[:keep] + "_" + suf
	}
	return out
}

func loadOrInitManifest(workDir, srcWS, tgtWS, srcRoot string, vers []string, excluded []string, retry bool, forceRefresh bool) (*state.Manifest, error) {
	path := filepath.Join(workDir, state.ManifestFile)
	if _, err := os.Stat(path); err != nil {
		return initManifest(workDir, srcWS, tgtWS, srcRoot, vers, excluded)
	}
	m, err := state.LoadManifest(workDir)
	if err != nil {
		return nil, err
	}
	if m.SourceWorkspaceID != srcWS || m.TargetWorkspaceID != tgtWS || m.SourceRootID != srcRoot {
		return nil, fmt.Errorf("manifest workspace/root mismatch: delete work-dir or use matching flags")
	}
	if !scope.StringSlicesEqual(m.DesiredVersions, vers) {
		return nil, fmt.Errorf("manifest versions mismatch: delete work-dir or use matching --versions")
	}
	if !scope.StringSlicesEqual(m.ExcludedPackages, excluded) {
		return nil, fmt.Errorf("manifest exclude-packages mismatch: delete work-dir or use matching --exclude-packages")
	}
	if forceRefresh {
		m.FetchComplete = false
		for i := range m.Items {
			m.Items[i].FetchDone = false
			m.Items[i].FetchError = ""
		}
	}
	if retry {
		// keep fetch state
	}
	return m, nil
}

func initManifest(workDir, srcWS, tgtWS, srcRoot string, vers []string, excluded []string) (*state.Manifest, error) {
	m := state.NewManifest(srcWS, tgtWS, srcRoot, vers, excluded)
	if err := state.SaveManifest(workDir, m); err != nil {
		return nil, err
	}
	return m, nil
}

func listDescendantPackages(c *apihub.Client, rootParentID, workspace string) ([]string, error) {
	var all []string
	page := 0
	for {
		list, _, err := c.ListPackagesPage(rootParentID, true, "package", page, 100)
		if err != nil {
			return nil, err
		}
		for _, p := range list.Packages {
			if p.Kind == "package" && underWorkspace(p.PackageId, workspace) {
				all = append(all, p.PackageId)
			}
		}
		if len(list.Packages) < 100 {
			break
		}
		page++
	}
	return all, nil
}

func filterExcludedPackages(pkgs []string, excluded []string) []string {
	var out []string
	for _, p := range pkgs {
		if scope.PackageExcluded(p, excluded) {
			continue
		}
		out = append(out, p)
	}
	return out
}

func underWorkspace(packageID, workspace string) bool {
	return packageID == workspace || strings.HasPrefix(packageID, workspace+".")
}

func runFetch(c *apihub.Client, m *state.Manifest, srcWS, srcRoot string, versionSpec []string, excluded []string, workDir string, force bool) error {
	var pkgs []string
	switch {
	case srcRoot == scope.WorkspaceRootSentinel:
		var err error
		pkgs, err = listDescendantPackages(c, srcWS, srcWS)
		if err != nil {
			return err
		}
	default:
		root, _, err := c.GetPackage(srcRoot)
		if err != nil {
			return fmt.Errorf("get source root: %w", err)
		}
		if !underWorkspace(srcRoot, srcWS) {
			return fmt.Errorf("source root %q is not under workspace %q", srcRoot, srcWS)
		}
		switch root.Kind {
		case "package":
			pkgs = []string{srcRoot}
		case "group":
			var err error
			pkgs, err = listDescendantPackages(c, srcRoot, srcWS)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("source root kind %q: expected package or group", root.Kind)
		}
	}

	before := len(pkgs)
	pkgs = filterExcludedPackages(pkgs, excluded)
	if before > 0 && len(pkgs) == 0 {
		logx.Warnf("all %d package(s) excluded by --exclude-packages; nothing to fetch", before)
	}
	sort.Strings(pkgs)

	m.Items = nil
	for _, p := range pkgs {
		var versionsToCopy []string
		if scope.IsAllVersions(versionSpec) {
			vlist, _, err := c.ListVersions(p)
			if err != nil {
				return fmt.Errorf("list versions %s: %w", p, err)
			}
			for _, pv := range vlist.Versions {
				versionsToCopy = append(versionsToCopy, pv.Version)
			}
			sort.Strings(versionsToCopy)
			if len(versionsToCopy) == 0 {
				logx.Warnf("package %s has no published versions — skip", p)
			}
		} else {
			versionsToCopy = append([]string(nil), versionSpec...)
		}
		for _, v := range versionsToCopy {
			m.Items = append(m.Items, state.WorkItem{
				SourcePackageID: p,
				Version:         v,
				Subdir:          dataSubdir(p, v),
			})
		}
	}

	for i := range m.Items {
		it := &m.Items[i]
		if it.FetchDone && !force {
			continue
		}
		vlist, _, err := c.ListVersions(it.SourcePackageID)
		if err != nil {
			it.FetchDone = false
			it.FetchError = err.Error()
			logx.Warnf("list versions %s: %v", it.SourcePackageID, err)
			continue
		}
		apiNames := make([]string, len(vlist.Versions))
		for i := range vlist.Versions {
			apiNames[i] = vlist.Versions[i].Version
		}
		resolved, ok := versionid.PickPublished(it.Version, apiNames)
		if !ok {
			logx.Warnf("package %s has no version %q — skip fetch", it.SourcePackageID, it.Version)
			if logx.IsDebug() {
				debugLogVersionMismatch(it.SourcePackageID, it.Version, vlist.Versions)
			}
			it.FetchDone = false
			it.FetchError = "version not found"
			continue
		}
		if resolved != it.Version {
			if logx.IsDebug() {
				logx.Debugf("%s: version spec %q resolved to API id %q", it.SourcePackageID, it.Version, resolved)
			}
			it.Version = resolved
			it.Subdir = dataSubdir(it.SourcePackageID, resolved)
		}
		zipb, _, err := c.GetVersionSources(it.SourcePackageID, it.Version)
		if err != nil {
			it.FetchError = err.Error()
			return fmt.Errorf("sources %s@%s: %w", it.SourcePackageID, it.Version, err)
		}
		cfg, _, err := c.GetVersionBuildConfigJSON(it.SourcePackageID, it.Version)
		if err != nil {
			it.FetchError = err.Error()
			return fmt.Errorf("config %s@%s: %w", it.SourcePackageID, it.Version, err)
		}
		base := filepath.Join(workDir, "data", it.Subdir)
		if err := os.MkdirAll(base, 0755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(base, "sources.zip"), zipb, 0644); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(base, "config.json"), cfg, 0644); err != nil {
			return err
		}
		it.FetchDone = true
		it.FetchError = ""
		if err := state.SaveManifest(workDir, m); err != nil {
			return err
		}
	}
	return nil
}

func runPlanLog(m *state.Manifest, workDir string) error {
	copied := manifestCopiedPkgVers(m)
	var items []copy.PkgVer
	configs := map[string][]byte{}
	for _, it := range m.Items {
		if !it.FetchDone {
			continue
		}
		cfgPath := filepath.Join(workDir, "data", it.Subdir, "config.json")
		b, err := os.ReadFile(cfgPath)
		if err != nil {
			return err
		}
		norm, err := copy.PrepareConfigSameScopePublish(b, it.SourcePackageID, copied)
		if err != nil {
			return err
		}
		items = append(items, copy.PkgVer{Pkg: it.SourcePackageID, Ver: it.Version})
		configs[copy.ItemCfgKey(it.SourcePackageID, it.Version)] = norm
	}
	order, warns, err := copy.TopologicalPublishOrder(items, configs)
	if err != nil {
		return err
	}
	logx.Section("Plan")
	logx.Step("Topological publish order")
	for i, nk := range order {
		logx.PlanItem(i+1, nk.Pkg, nk.Ver)
	}
	for _, w := range warns {
		logx.PlanWarn(w)
	}
	return nil
}

func collectStructure(c *apihub.Client, m *state.Manifest, srcWS string) (sorted []string, meta map[string]*apihub.PackageInfo, err error) {
	pkgSet := map[string]struct{}{}
	for _, it := range m.Items {
		if !it.FetchDone {
			continue
		}
		pkgSet[it.SourcePackageID] = struct{}{}
	}
	meta = map[string]*apihub.PackageInfo{}
	for pkg := range pkgSet {
		id := pkg
		for id != "" && id != srcWS {
			if _, ok := meta[id]; ok {
				break
			}
			pi, _, err := c.GetPackage(id)
			if err != nil {
				return nil, nil, err
			}
			meta[id] = pi
			id = pi.ParentId
		}
	}
	for id := range meta {
		sorted = append(sorted, id)
	}
	sort.Slice(sorted, func(i, j int) bool {
		di := strings.Count(sorted[i], ".")
		dj := strings.Count(sorted[j], ".")
		if di != dj {
			return di < dj
		}
		return sorted[i] < sorted[j]
	})
	return sorted, meta, nil
}

func runApply(srcCl, tgtCl *apihub.Client, m *state.Manifest, workDir, srcWS, tgtWS string, retry bool) error {
	if m.PackageMap == nil {
		m.PackageMap = map[string]string{}
	}
	m.PackageMap[srcWS] = tgtWS

	sorted, meta, err := collectStructure(srcCl, m, srcWS)
	if err != nil {
		return err
	}

	for _, sid := range sorted {
		if sid == srcWS {
			continue
		}
		if _, ok := m.PackageMap[sid]; ok {
			continue
		}
		pi := meta[sid]
		srcParent := pi.ParentId
		tgtParent, ok := m.PackageMap[srcParent]
		if !ok {
			return fmt.Errorf("internal: no target mapping for parent %q of %q", srcParent, sid)
		}
		body := apihub.CreatePackageBody{
			Alias:       pi.Alias,
			ParentId:    tgtParent,
			Kind:        pi.Kind,
			Name:        pi.Name,
			Description: pi.Description,
		}
		created, code, err := tgtCl.CreatePackage(body)
		if err != nil {
			return fmt.Errorf("create %q: %w", sid, err)
		}
		if code != 201 {
			return fmt.Errorf("create %q: unexpected code %d", sid, code)
		}
		m.PackageMap[sid] = created.PackageId
		logx.Okf(`created package %s → target id %s`, sid, created.PackageId)
		if err := state.SaveManifest(workDir, m); err != nil {
			return err
		}
	}

	copied := manifestCopiedPkgVers(m)

	var items []copy.PkgVer
	configs := map[string][]byte{}
	for _, it := range m.Items {
		if !it.FetchDone {
			continue
		}
		cfgPath := filepath.Join(workDir, "data", it.Subdir, "config.json")
		b, err := os.ReadFile(cfgPath)
		if err != nil {
			return err
		}
		norm, err := copy.PrepareConfigSameScopePublish(b, it.SourcePackageID, copied)
		if err != nil {
			return err
		}
		items = append(items, copy.PkgVer{Pkg: it.SourcePackageID, Ver: it.Version})
		configs[copy.ItemCfgKey(it.SourcePackageID, it.Version)] = norm
	}
	order, _, err := copy.TopologicalPublishOrder(items, configs)
	if err != nil {
		return err
	}

	for _, nk := range order {
		var wi *state.WorkItem
		for i := range m.Items {
			if m.Items[i].SourcePackageID == nk.Pkg && m.Items[i].Version == nk.Ver {
				wi = &m.Items[i]
				break
			}
		}
		if wi == nil {
			continue
		}
		if wi.Published && retry && wi.PublishErr == "" {
			continue
		}
		if wi.Published && !retry {
			continue
		}
		srcPkg := nk.Pkg
		tgtPkg, ok := m.PackageMap[srcPkg]
		if !ok {
			return fmt.Errorf("no target id for source package %q", srcPkg)
		}
		cfgRaw, err := os.ReadFile(filepath.Join(workDir, "data", wi.Subdir, "config.json"))
		if err != nil {
			return err
		}
		zipRaw, err := os.ReadFile(filepath.Join(workDir, "data", wi.Subdir, "sources.zip"))
		if err != nil {
			return err
		}
		outCfg, err := copy.RemapBuildConfig(cfgRaw, srcPkg, tgtPkg, m.PackageMap, copied)
		if err != nil {
			wi.PublishErr = err.Error()
			continue
		}
		pid, sync, _, err := tgtCl.PublishVersion(tgtPkg, zipRaw, outCfg, true, true)
		if err != nil {
			logx.Errorf(`publish failed %s @ %s: %v`, tgtPkg, nk.Ver, err)
			wi.Published = false
			wi.PublishErr = err.Error()
			_ = state.SaveManifest(workDir, m)
			continue
		}
		wi.PublishID = pid
		if sync {
			wi.Published = true
			wi.PublishErr = ""
			logx.Okf(`published %s @ %s (sync)`, tgtPkg, nk.Ver)
			_ = state.SaveManifest(workDir, m)
			continue
		}
		if err := waitPublishDone(tgtCl, tgtPkg, pid); err != nil {
			logx.Errorf(`publish status %s @ %s (id=%s): %v`, tgtPkg, nk.Ver, pid, err)
			wi.Published = false
			wi.PublishErr = err.Error()
			_ = state.SaveManifest(workDir, m)
			continue
		}
		wi.Published = true
		wi.PublishErr = ""
		logx.Okf(`published %s @ %s`, tgtPkg, nk.Ver)
		if err := state.SaveManifest(workDir, m); err != nil {
			return err
		}
	}
	return nil
}

func manifestCopiedPkgVers(m *state.Manifest) []copy.PkgVer {
	var out []copy.PkgVer
	for _, it := range m.Items {
		if !it.FetchDone {
			continue
		}
		out = append(out, copy.PkgVer{Pkg: it.SourcePackageID, Ver: it.Version})
	}
	return out
}

func sortedDistinctVersionNames(pvs []apihub.PublishedVersionListView) []string {
	seen := make(map[string]struct{}, len(pvs))
	var out []string
	for _, pv := range pvs {
		v := pv.Version
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

func debugLogVersionMismatch(pkg, want string, pvs []apihub.PublishedVersionListView) {
	logx.Debugf("exact match on UTF-8 string %q (%d raw bytes)", want, len([]byte(want)))
	names := sortedDistinctVersionNames(pvs)
	if len(names) == 0 {
		logx.Debugf("merged GET /versions list is empty for %s (after paging)", pkg)
		return
	}
	const show = 50
	part := names
	tail := ""
	if len(part) > show {
		part = part[:show]
		tail = fmt.Sprintf("; …+%d omitted", len(names)-show)
	}
	var sb strings.Builder
	for i, w := range part {
		if i > 0 {
			sb.WriteString(", ")
		}
		fmt.Fprintf(&sb, "%q", w)
	}
	logx.Debugf("%s merged version strings (%d unique): %s%s", pkg, len(names), sb.String(), tail)
}

func waitPublishDone(c *apihub.Client, packageID, publishID string) error {
	deadline := time.Now().Add(2 * time.Hour)
	for time.Now().Before(deadline) {
		st, _, err := c.GetPublishStatus(packageID, publishID)
		if err != nil {
			return err
		}
		switch strings.ToLower(st.Status) {
		case "complete":
			return nil
		case "error":
			if st.Message != "" {
				return fmt.Errorf("%s", st.Message)
			}
			return fmt.Errorf("publish error")
		default:
			time.Sleep(2 * time.Second)
		}
	}
	return fmt.Errorf("publish timeout for %s", publishID)
}
