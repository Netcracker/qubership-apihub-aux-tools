// Package pipeline sequences the whole run: preflight → fetch → parse → merge
// → publish → groups → exports, writing artifacts and the report along the way.
package pipeline

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"apihub-ddl-import/internal/apihub"
	"apihub-ddl-import/internal/config"
	"apihub-ddl-import/internal/ddl"
	"apihub-ddl-import/internal/enrich"
	"apihub-ddl-import/internal/groups"
	"apihub-ddl-import/internal/logx"
	"apihub-ddl-import/internal/merge"
	"apihub-ddl-import/internal/model"
	"apihub-ddl-import/internal/report"
	"apihub-ddl-import/internal/source"
	"apihub-ddl-import/internal/xlsxin"
	"apihub-ddl-import/internal/zipbuild"
)

// PreviousVersionNone is the explicit "first publish, no baseline" value.
const PreviousVersionNone = "none"

// Exit codes.
const (
	ExitOK     = 0
	ExitFatal  = 1
	ExitStrict = 3
)

// Options are the resolved run parameters (flag > env > config > default).
type Options struct {
	Cfg             *config.Config
	Version         string
	PreviousVersion string
	Status          string
	DryRun          bool
	Strict          bool
	SkipGroups      bool
	SkipExports     bool
	InsecureTLS     bool
	PublishTimeout  time.Duration
	VersionLabels   []string
}

// Run executes the pipeline and returns the process exit code.
func Run(opts Options) int {
	cfg := opts.Cfg
	outDir := cfg.Output.Dir
	rep := report.New(report.Inputs{
		DDLSource:       describeSource(cfg.DDLSource),
		CommentsSource:  describeSource(cfg.CommentsSource),
		PackageID:       cfg.Apihub.PackageID,
		Version:         opts.Version,
		PreviousVersion: opts.PreviousVersion,
		Status:          opts.Status,
		DryRun:          opts.DryRun,
	})
	fail := func(fe *model.FatalError) int {
		rep.SetFatal(fe)
		rep.PrintSummary()
		if err := rep.WriteFiles(outDir); err != nil {
			logx.Errorf("write report: %v", err)
		}
		return ExitFatal
	}

	var client *apihub.Client
	if !opts.DryRun {
		client = apihub.New(cfg.Apihub.URL, cfg.Apihub.APIKey, opts.InsecureTLS)
		if code := preflight(client, cfg.Apihub.PackageID, opts, fail); code != ExitOK {
			return code
		}
	}
	source.InsecureTLS = opts.InsecureTLS

	// ---- Fetch sources.
	logx.Section("Fetch sources")
	logx.Infof("DDL from %s", describeSource(cfg.DDLSource))
	ddlFiles, err := source.FetchDDL(cfg.DDLSource)
	if err != nil {
		return fail(model.Fatalf(model.FNoDdlFiles, "fetch DDL: %v", err))
	}
	if len(ddlFiles) == 0 {
		return fail(model.Fatalf(model.FNoDdlFiles, "DDL source %s contains no .sql files", describeSource(cfg.DDLSource)))
	}
	logx.Okf("%d .sql file(s)", len(ddlFiles))
	logx.Infof("comments from %s", describeSource(cfg.CommentsSource))
	commentsData, commentsName, err := source.FetchComments(cfg.CommentsSource)
	if err != nil {
		return fail(model.Fatalf(model.FXlsxHeader, "fetch comments workbook: %v", err))
	}
	logx.Okf("workbook %s (%d bytes)", commentsName, len(commentsData))
	saveRaw(outDir, ddlFiles, commentsData, commentsName)

	// ---- Parse.
	logx.Section("Parse")
	doc, err := xlsxin.Read(commentsData)
	if err != nil {
		var fe *model.FatalError
		if errors.As(err, &fe) {
			return fail(fe)
		}
		return fail(model.Fatalf(model.FXlsxHeader, "%v", err))
	}
	logx.Okf("workbook: %d table rows, %d column rows", len(doc.Tables), len(doc.Columns))
	ddlm, err := ddl.ParseFiles(ddlFiles)
	if err != nil {
		var fe *model.FatalError
		if errors.As(err, &fe) {
			return fail(fe)
		}
		return fail(model.Fatalf(model.FDdlParse, "%v", err))
	}
	cols := 0
	for _, t := range ddlm.Tables {
		cols += len(t.Columns)
	}
	logx.Okf("DDL: %d tables, %d columns", len(ddlm.Tables), cols)

	// ---- Merge + enriched artifacts + report.
	logx.Section("Merge")
	res := merge.Merge(ddlm, doc)
	enriched := ddl.RenderEnriched(ddlm, res)
	for rel, content := range enriched {
		p := filepath.Join(outDir, "enriched", filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err == nil {
			err = os.WriteFile(p, []byte(content), 0o644)
		}
		if err != nil {
			return fail(model.Fatalf(model.FNoDdlFiles, "write enriched %s: %v", rel, err))
		}
	}
	logx.Okf("enriched DDL written to %s", filepath.Join(outDir, "enriched"))
	rep.SetMerge(res)
	rep.PrintSummary()
	if err := rep.WriteFiles(outDir); err != nil {
		logx.Errorf("write report: %v", err)
	}
	logx.Notef("report: %s, %s", filepath.Join(outDir, "report.md"), filepath.Join(outDir, "report.json"))

	if opts.Strict && len(res.Warnings) > 0 {
		logx.Warnf("--strict: %d warning(s) present, stopping before publish", len(res.Warnings))
		return ExitStrict
	}
	if opts.DryRun {
		logx.Okf("dry-run complete, no APIHUB calls made")
		return ExitOK
	}

	// ---- Publish.
	if code := publish(client, cfg, opts, enriched, rep, fail); code != ExitOK {
		return code
	}

	// ---- Post-publish: entities gate.
	logx.Section("Verify published DDL entities")
	entities, err := client.ListDdlEntities(cfg.Apihub.PackageID, opts.Version)
	if err != nil {
		return fail(model.Fatalf(model.FNoDdlEntities, "list DDL entities: %v", err))
	}
	if len(entities) == 0 {
		return fail(model.Fatalf(model.FNoDdlEntities,
			"version %s published but has zero DDL entities — the APIHUB builder likely does not parse .sql sources into DDL contracts; check the builder version", opts.Version))
	}
	logx.Okf("%d DDL entities published", len(entities))
	withDesc := 0
	for _, e := range entities {
		if strings.TrimSpace(e.Description) != "" {
			withDesc++
		}
	}
	if withDesc == 0 && res.Stats.TableCommentsGenerated > 0 {
		rep.AddWarning(model.Warning{Code: model.WDescriptionsMissing,
			Msg: "no published DDL entity carries a description although COMMENT ON statements were generated — the builder may ignore COMMENT ON"})
		logx.Warnf("published entities have no descriptions — builder may ignore COMMENT ON")
	}

	// ---- Groups by domain.
	if opts.SkipGroups || !cfg.GroupsEnabled() {
		rep.Groups = &report.GroupsInfo{Skipped: true, APIAvailable: true}
		logx.Infof("groups step skipped")
	} else {
		logx.Section("DDL table groups")
		out := groups.Create(client, cfg.Apihub.PackageID, opts.Version, res, entities, cfg.Groups.DescriptionTemplate)
		rep.Groups = &report.GroupsInfo{
			APIAvailable: out.APIAvailable,
			Created:      out.Created,
			Updated:      out.Updated,
			Failed:       out.Failed,
		}
		for _, w := range out.Warnings {
			rep.AddWarning(w)
		}
	}

	// ---- Exports.
	if opts.SkipExports {
		logx.Infof("exports step skipped")
	} else {
		runExports(client, cfg, opts, res, entities, rep)
	}

	if err := rep.WriteFiles(outDir); err != nil {
		logx.Errorf("write report: %v", err)
	}
	logx.Done()
	return ExitOK
}

func preflight(client *apihub.Client, packageID string, opts Options, fail func(*model.FatalError) int) int {
	logx.Section("Preflight")
	if _, found, err := client.GetVersion(packageID, opts.Version); err != nil {
		return fail(model.Fatalf(model.FApihubUnreachable, "check version %s: %v", opts.Version, err))
	} else if found {
		logx.Warnf("version %s already exists in %s — publishing will add a new revision", opts.Version, packageID)
	}
	if opts.PreviousVersion == PreviousVersionNone {
		logx.Infof("previous version: none (first publish, no changelog baseline)")
		return ExitOK
	}
	info, found, err := client.GetVersion(packageID, opts.PreviousVersion)
	if err != nil {
		return fail(model.Fatalf(model.FApihubUnreachable, "check previous version %s: %v", opts.PreviousVersion, err))
	}
	if !found {
		return fail(model.Fatalf(model.FPrevVersionNotFound,
			"previous version %q is not published in package %s", opts.PreviousVersion, packageID))
	}
	logx.Okf("previous version %s exists (status %s)", opts.PreviousVersion, info.Status)
	return ExitOK
}

func publish(client *apihub.Client, cfg *config.Config, opts Options, enriched map[string]string, rep *report.Report, fail func(*model.FatalError) int) int {
	logx.Section("Publish to APIHUB")
	fileIDs := make([]string, 0, len(enriched))
	for rel := range enriched {
		fileIDs = append(fileIDs, rel)
	}
	zipBytes, err := zipbuild.SourcesZip(enriched)
	if err != nil {
		return fail(model.Fatalf(model.FPublishFailed, "build sources zip: %v", err))
	}
	cfgJSON, err := zipbuild.ConfigJSON(cfg.Apihub.PackageID, opts.Version, opts.Status, opts.PreviousVersion, fileIDs, opts.VersionLabels)
	if err != nil {
		return fail(model.Fatalf(model.FPublishFailed, "build config: %v", err))
	}
	logx.Infof("publishing %s@%s (status %s, %d files, %d KB sources)",
		cfg.Apihub.PackageID, opts.Version, opts.Status, len(fileIDs), len(zipBytes)/1024)
	start := time.Now()
	publishID, sync, err := client.PublishVersion(cfg.Apihub.PackageID, zipBytes, cfgJSON)
	if err != nil {
		return fail(model.Fatalf(model.FPublishFailed, "publish: %v", err))
	}
	pub := &report.PublishInfo{PublishID: publishID}
	rep.Publish = pub
	if sync {
		pub.Status = "complete"
		logx.Okf("publish accepted synchronously (204)")
		return ExitOK
	}
	logx.Infof("publishId %s, polling status (timeout %s)…", publishID, opts.PublishTimeout)
	deadline := time.Now().Add(opts.PublishTimeout)
	for {
		st, err := client.GetPublishStatus(cfg.Apihub.PackageID, publishID)
		if err != nil {
			pub.Status = "error"
			pub.Message = err.Error()
			return fail(model.Fatalf(model.FPublishFailed, "publish status: %v", err))
		}
		pub.Status = st.Status
		pub.Message = st.Message
		switch st.Status {
		case "complete":
			pub.DurationSec = time.Since(start).Seconds()
			logx.Okf("publish complete in %.1fs", pub.DurationSec)
			return ExitOK
		case "error":
			pub.DurationSec = time.Since(start).Seconds()
			return fail(model.Fatalf(model.FPublishFailed, "publish failed: %s", st.Message))
		}
		if time.Now().After(deadline) {
			return fail(model.Fatalf(model.FPublishFailed,
				"publish %s still %q after %s — check the build queue", publishID, st.Status, opts.PublishTimeout))
		}
		time.Sleep(2 * time.Second)
	}
}

func runExports(client *apihub.Client, cfg *config.Config, opts Options, res *model.MergeResult, entities []apihub.DdlEntity, rep *report.Report) {
	logx.Section("Exports")
	exDir := filepath.Join(cfg.Output.Dir, "export")
	if err := os.MkdirAll(exDir, 0o755); err != nil {
		logx.Errorf("create %s: %v", exDir, err)
		return
	}
	rep.Exports = &report.ExportsInfo{}

	var dataTypeRe *regexp.Regexp
	if cfg.Analytics.DataTypeChangeRegex != "" {
		re, err := regexp.Compile(cfg.Analytics.DataTypeChangeRegex)
		if err != nil {
			logx.Warnf("invalid analytics.dataTypeChangeRegex %q, using default: %v", cfg.Analytics.DataTypeChangeRegex, err)
		} else {
			dataTypeRe = re
		}
	}
	idByName := make(map[string]string, len(entities))
	for _, e := range entities {
		idByName[model.NormKey(e.Name)] = e.DdlEntityId
	}
	entOpts := enrich.Options{DomainByTable: res.DomainByTable}

	// Entities export.
	if raw, err := client.ExportDdlEntities(cfg.Apihub.PackageID, opts.Version); err != nil {
		rep.AddWarning(model.Warning{Code: model.WExportUnavailable, Msg: fmt.Sprintf("entities export: %v", err)})
		logx.Warnf("entities export unavailable: %v", err)
	} else {
		name := exportFileName("DDLEntities", cfg.Apihub.PackageID, opts.Version)
		saveExport(exDir, name, raw)
		out, info, err := enrich.Entities(raw, entOpts)
		if err != nil {
			logx.Errorf("enrich entities export: %v", err)
		} else {
			writeExport(exDir, name, out)
			rep.Exports.Entities = exportInfo(filepath.Join(exDir, name), info)
			for _, w := range info.Warnings {
				rep.AddWarning(w)
			}
			logx.Okf("entities export: %s (%d rows, Group filled in %d)", name, info.Rows, info.GroupFilled)
		}
	}

	// Changes export (needs a baseline).
	if opts.PreviousVersion == PreviousVersionNone {
		logx.Infof("changes export skipped (no previous version)")
		return
	}
	raw, err := client.ExportDdlChanges(cfg.Apihub.PackageID, opts.Version, opts.PreviousVersion, "")
	if err != nil {
		rep.AddWarning(model.Warning{Code: model.WExportUnavailable, Msg: fmt.Sprintf("changes export: %v", err)})
		logx.Warnf("changes export unavailable: %v", err)
		return
	}
	name := exportFileName("DDLChanges", cfg.Apihub.PackageID, opts.Version)
	saveExport(exDir, name, raw)
	chOpts := enrich.Options{
		DomainByTable:  res.DomainByTable,
		EntityIDByName: idByName,
		DataTypeRegex:  dataTypeRe,
		FetchChanges: func(id string) ([]enrich.Change, error) {
			chs, err := client.GetDdlEntityChanges(cfg.Apihub.PackageID, opts.Version, id, opts.PreviousVersion, "")
			if err != nil {
				return nil, err
			}
			out := make([]enrich.Change, 0, len(chs))
			for _, c := range chs {
				out = append(out, enrich.Change{Description: c.Description, Severity: c.Severity})
			}
			return out, nil
		},
	}
	out, info, err := enrich.Changes(raw, chOpts)
	if err != nil {
		logx.Errorf("enrich changes export: %v", err)
		return
	}
	writeExport(exDir, name, out)
	rep.Exports.Changes = exportInfo(filepath.Join(exDir, name), info)
	for _, w := range info.Warnings {
		rep.AddWarning(w)
	}
	if info.AnalyticsFallback {
		rep.AddWarning(model.Warning{Code: model.WAnalyticsFallback,
			Msg: "Analytics Severity for some rows was derived from the severity count columns (per-change data unavailable); the data-type exception cannot be applied there"})
	}
	logx.Okf("changes export: %s (%d rows, Group filled in %d)", name, info.Rows, info.GroupFilled)
}

func exportInfo(path string, info *enrich.Info) *report.ExportFile {
	return &report.ExportFile{
		Path:              path,
		Rows:              info.Rows,
		GroupFilled:       info.GroupFilled,
		GroupAppended:     info.GroupAppended,
		AnalyticsFallback: info.AnalyticsFallback,
	}
}

var unsafeFileChars = regexp.MustCompile(`[\\/:*?"<>|\s]+`)

func exportFileName(kind, packageID, version string) string {
	return fmt.Sprintf("%s_%s_%s.xlsx", kind,
		unsafeFileChars.ReplaceAllString(packageID, "_"),
		unsafeFileChars.ReplaceAllString(version, "_"))
}

func saveExport(dir, name string, raw []byte) {
	orig := strings.TrimSuffix(name, ".xlsx") + ".orig.xlsx"
	if err := os.WriteFile(filepath.Join(dir, orig), raw, 0o644); err != nil {
		logx.Errorf("save %s: %v", orig, err)
	}
}

func writeExport(dir, name string, data []byte) {
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
		logx.Errorf("save %s: %v", name, err)
	}
}

func saveRaw(outDir string, ddlFiles []model.File, comments []byte, commentsName string) {
	for _, f := range ddlFiles {
		p := filepath.Join(outDir, "raw", "ddl", filepath.FromSlash(f.RelPath))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err == nil {
			_ = os.WriteFile(p, f.Data, 0o644)
		}
	}
	p := filepath.Join(outDir, "raw", commentsName)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err == nil {
		_ = os.WriteFile(p, comments, 0o644)
	}
}

func describeSource(s config.Source) string {
	if s.Type == config.SourceGitlab {
		return fmt.Sprintf("gitlab %s @ %s : %s", s.Repo, s.Branch, s.Path)
	}
	return fmt.Sprintf("file %s", s.Path)
}
