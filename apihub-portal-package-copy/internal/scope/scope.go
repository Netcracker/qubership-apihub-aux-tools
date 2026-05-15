package scope

import (
	"fmt"
	"sort"
	"strings"
)

const AllVersionsSentinel = "*"
const WorkspaceRootSentinel = "*"

// QualifyUnderWorkspace resolves a Portal package/group id relative to workspace.
//
// Returns id unchanged when it equals workspace, already has prefix workspace+".", or is the sentinel "*".
// Otherwise returns workspace+"."+id so a short suffix (e.g. "BSS") is interpreted under the workspace
// previously passed separately (composite ids like WORKSPACE.CHILD.CHILD).
func QualifyUnderWorkspace(workspace, id string) string {
	workspace = strings.TrimSpace(workspace)
	id = strings.TrimSpace(id)
	if workspace == "" || id == "" {
		return id
	}
	if id == WorkspaceRootSentinel {
		return id
	}
	if id == workspace || strings.HasPrefix(id, workspace+".") {
		return id
	}
	return workspace + "." + id
}

// QualifyExcludedUnderWorkspace qualifies each exclusion entry relative to workspace (same rules as QualifyUnderWorkspace).
func QualifyExcludedUnderWorkspace(workspace string, exclusions []string) []string {
	if len(exclusions) == 0 {
		return nil
	}
	out := make([]string, 0, len(exclusions))
	for _, raw := range exclusions {
		ex := strings.TrimSpace(raw)
		if ex == "" {
			continue
		}
		out = append(out, QualifyUnderWorkspace(workspace, ex))
	}
	return DedupSorted(out)
}

// NormalizeVersionTokens returns canonical version spec: either []string{"*"} or trimmed non-empty names.
func NormalizeVersionTokens(raw []string) ([]string, error) {
	var out []string
	var sawStar bool
	for _, x := range raw {
		x = strings.TrimSpace(x)
		if x == "" {
			continue
		}
		if x == AllVersionsSentinel {
			sawStar = true
			continue
		}
		out = append(out, x)
	}
	if sawStar && len(out) > 0 {
		return nil, fmt.Errorf("versions: use %q alone to copy every version, or list explicit versions — do not combine with other entries", AllVersionsSentinel)
	}
	if sawStar {
		return []string{AllVersionsSentinel}, nil
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("versions: empty after parsing")
	}
	return out, nil
}

// IsAllVersions reports whether spec is the single "*" mode.
func IsAllVersions(vers []string) bool {
	return len(vers) == 1 && vers[0] == AllVersionsSentinel
}

// PackageExcluded reports true if packageID should be skipped because it matches exclusion
// rule (exact id or descendant of excluded group/package path).
func PackageExcluded(packageID string, exclusions []string) bool {
	for _, raw := range exclusions {
		ex := strings.TrimSpace(raw)
		if ex == "" {
			continue
		}
		if packageID == ex {
			return true
		}
		if strings.HasPrefix(packageID, ex+".") {
			return true
		}
	}
	return false
}

// DedupSorted returns a sorted deduplicated copy.
func DedupSorted(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	var out []string
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// StringSlicesEqual ignores order when both slices are sorted copies of DedupSorted.
func StringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	aa := append([]string(nil), a...)
	bb := append([]string(nil), b...)
	sort.Strings(aa)
	sort.Strings(bb)
	for i := range aa {
		if aa[i] != bb[i] {
			return false
		}
	}
	return true
}
