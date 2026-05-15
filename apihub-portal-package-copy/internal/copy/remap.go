package copy

import (
	"encoding/json"
	"fmt"
	"strings"

	"apihub-portal-package-copy/internal/versionid"
)

// SourceVersionInCopySet matches refVer (as stored in publish config, often base-only like "2025.1")
// against the resolved catalogue ids in copied (e.g. "2025.1@3").
func SourceVersionInCopySet(copied []PkgVer, sourcePkg, refVer string) bool {
	refVer = strings.TrimSpace(refVer)
	if refVer == "" || len(copied) == 0 {
		return false
	}
	for _, it := range copied {
		if it.Pkg != sourcePkg {
			continue
		}
		if versionid.SpecMatchesPublished(refVer, it.Ver) {
			return true
		}
	}
	return false
}

func clearPreviousUnlessCopied(cfg map[string]interface{}, owningSourcePackageID string, copied []PkgVer) {
	if len(copied) == 0 {
		return
	}
	pv, ok := cfg["previousVersion"].(string)
	if !ok {
		return
	}
	ref := strings.TrimSpace(pv)
	if ref == "" {
		return
	}
	prevPkg := owningSourcePackageID
	if p, ok := cfg["previousVersionPackageId"].(string); ok {
		if t := strings.TrimSpace(p); t != "" {
			prevPkg = t
		}
	}
	if !SourceVersionInCopySet(copied, prevPkg, ref) {
		cfg["previousVersion"] = ""
		cfg["previousVersionPackageId"] = ""
	}
}

// PrepareConfigSameScopePublish returns JSON normalized for in-memory graph/publish: drops previousVersion
// links that point at versions not included in this copy run (source-side package ids and resolved version rows).
func PrepareConfigSameScopePublish(raw []byte, owningSourcePackageID string, copied []PkgVer) ([]byte, error) {
	var cfg map[string]interface{}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	clearPreviousUnlessCopied(cfg, owningSourcePackageID, copied)
	return json.Marshal(cfg)
}

// RemapBuildConfig adjusts stored publish JSON for the target package and remapped dependency ids.
// copied lists every (sourcePackageId, resolvedVersion) row being copied; used to drop stale previousVersion edges.
func RemapBuildConfig(raw []byte, sourcePackageID, targetPackageID string, idMap map[string]string, copied []PkgVer) ([]byte, error) {
	var cfg map[string]interface{}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}

	clearPreviousUnlessCopied(cfg, sourcePackageID, copied)

	remapIfMapped := func(id string) string {
		if id == "" {
			return ""
		}
		if id == sourcePackageID {
			return targetPackageID
		}
		if nv, ok := idMap[id]; ok {
			return nv
		}
		return id
	}

	cfg["packageId"] = targetPackageID

	switch pv := cfg["previousVersionPackageId"].(type) {
	case string:
		if pv == "" {
			// leave empty (same-package semantics on backend)
		} else {
			cfg["previousVersionPackageId"] = remapIfMapped(pv)
		}
	default:
		// omit or nil
	}

	if refs, ok := cfg["refs"].([]interface{}); ok {
		for _, r := range refs {
			rm, ok := r.(map[string]interface{})
			if !ok {
				continue
			}
			if v, ok := rm["refId"].(string); ok {
				rm["refId"] = remapIfMapped(v)
			}
			if v, ok := rm["parentRefId"].(string); ok && v != "" {
				rm["parentRefId"] = remapIfMapped(v)
			}
		}
	}

	for _, k := range []string{
		"publishId", "createdBy", "migrationId", "migrationBuild",
		"publishedAt", "noChangeLog",
	} {
		delete(cfg, k)
	}

	// Backend assigns revision ("@n") on publish; /config snapshots use full ids (e.g. 2025.4@1).
	if v, ok := cfg["version"].(string); ok {
		v = strings.TrimSpace(v)
		if v != "" {
			cfg["version"] = versionid.BaseOnly(v)
		}
	}
	if v, ok := cfg["previousVersion"].(string); ok {
		cfg["previousVersion"] = versionid.BaseOnly(strings.TrimSpace(v))
	}

	return json.Marshal(cfg)
}

type prevEdge struct {
	PrevVersion string `json:"previousVersion"`
	PrevPkg     string `json:"previousVersionPackageId"`
}

// PreviousPointer extracts previous-version link from publish config JSON.
func PreviousPointer(raw []byte, defaultPkg string) (prevPkg, prevVer string, err error) {
	var pe prevEdge
	if err := json.Unmarshal(raw, &pe); err != nil {
		return "", "", err
	}
	if pe.PrevVersion == "" {
		return "", "", nil
	}
	pkg := pe.PrevPkg
	if pkg == "" {
		pkg = defaultPkg
	}
	return pkg, pe.PrevVersion, nil
}

type nodeKey struct {
	Pkg string
	Ver string
}

// ResolveCopiedVersionKey maps a stored previousVersion string (often base-only like "2025.1")
// to the canonical row id in items (e.g. "2025.1@3").
func ResolveCopiedVersionKey(sourcePkg, prevRef string, items []PkgVer) (resolved string, ok bool) {
	prevRef = strings.TrimSpace(prevRef)
	for _, it := range items {
		if it.Pkg != sourcePkg {
			continue
		}
		if versionid.SpecMatchesPublished(prevRef, it.Ver) {
			return it.Ver, true
		}
	}
	return "", false
}

// PkgVer identifies one copied package version.
type PkgVer struct {
	Pkg string
	Ver string
}

// TopologicalPublishOrder returns an order of (packageId, version) respecting previousVersion edges between copied nodes.
func TopologicalPublishOrder(items []PkgVer, configs map[string][]byte) (order []nodeKey, warnings []string, err error) {
	nodes := map[nodeKey]bool{}
	for _, it := range items {
		nodes[nodeKey{it.Pkg, it.Ver}] = true
	}

	type edge struct {
		from nodeKey
		to   nodeKey
	}
	var edges []edge

	for _, it := range items {
		key := nodeKey{it.Pkg, it.Ver}
		raw := configs[ItemCfgKey(it.Pkg, it.Ver)]
		if len(raw) == 0 {
			continue
		}
		prevPkg, prevVerRef, err := PreviousPointer(raw, it.Pkg)
		if err != nil {
			return nil, nil, fmt.Errorf("%s@%s: %w", it.Pkg, it.Ver, err)
		}
		prevVerRef = strings.TrimSpace(prevVerRef)
		if prevVerRef == "" {
			continue
		}
		prevResolved, ok := ResolveCopiedVersionKey(prevPkg, prevVerRef, items)
		if !ok {
			warnings = append(warnings, fmt.Sprintf("version %s@%s references previous %s@%s which is not in the copy set", it.Pkg, it.Ver, prevPkg, prevVerRef))
			continue
		}
		from := nodeKey{prevPkg, prevResolved}
		edges = append(edges, edge{from: from, to: key})
	}

	indeg := map[nodeKey]int{}
	for n := range nodes {
		indeg[n] = 0
	}
	adj := map[nodeKey][]nodeKey{}
	for _, e := range edges {
		adj[e.from] = append(adj[e.from], e.to)
		indeg[e.to]++
	}

	var q []nodeKey
	for n, d := range indeg {
		if d == 0 {
			q = append(q, n)
		}
	}

	sortNodes := func(a []nodeKey) {
		for i := 0; i < len(a); i++ {
			for j := i + 1; j < len(a); j++ {
				if a[j].Pkg < a[i].Pkg || (a[j].Pkg == a[i].Pkg && a[j].Ver < a[i].Ver) {
					a[i], a[j] = a[j], a[i]
				}
			}
		}
	}
	sortNodes(q)

	var out []nodeKey
	for len(q) > 0 {
		n := q[0]
		q = q[1:]
		out = append(out, n)
		for _, m := range adj[n] {
			indeg[m]--
			if indeg[m] == 0 {
				q = append(q, m)
				sortNodes(q)
			}
		}
	}
	if len(out) != len(nodes) {
		return nil, warnings, fmt.Errorf("cycle detected in version previousVersion graph")
	}
	return out, warnings, nil
}

// ItemCfgKey joins package and version for map keys.
func ItemCfgKey(pkg, ver string) string {
	return pkg + "\x00" + ver
}
