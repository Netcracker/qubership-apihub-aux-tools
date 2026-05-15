// Package versionid handles Portal composite version identifiers (logical name + revision suffix "@k").
package versionid

import (
	"sort"
	"strconv"
	"strings"
)

// SplitBaseRevision splits the last "@suffix" revision segment. If no "@", base is v and ok is false.
func SplitBaseRevision(v string) (base string, rev string, ok bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", "", false
	}
	i := strings.LastIndex(v, "@")
	if i < 0 {
		return v, "", false
	}
	return strings.TrimSpace(v[:i]), strings.TrimSpace(v[i+1:]), true
}

// BaseOnly returns the logical version without a revision suffix (strip last "@suffix").
func BaseOnly(v string) string {
	b, _, _ := SplitBaseRevision(v)
	if b == "" {
		return strings.TrimSpace(v)
	}
	return b
}

// SpecMatchesPublished reports whether a user-facing version spec matches an API list entry.
// If spec contains "@", only exact string equality matches. Otherwise spec matches apiVersion
// when apiVersion equals spec or apiVersion is spec + "@" + revision (e.g. 2025.4 matches 2025.4@1).
func SpecMatchesPublished(spec, apiVersion string) bool {
	spec = strings.TrimSpace(spec)
	apiVersion = strings.TrimSpace(apiVersion)
	if spec == "" || apiVersion == "" {
		return false
	}
	if strings.Contains(spec, "@") {
		return spec == apiVersion
	}
	if apiVersion == spec {
		return true
	}
	return strings.HasPrefix(apiVersion, spec+"@")
}

// PickPublished picks one API version string for a user spec. If several rows match the same base
// (unusual when the API returns only latest revisions), the highest numeric revision wins.
func PickPublished(userSpec string, apiVersions []string) (resolved string, ok bool) {
	var hits []string
	for _, v := range apiVersions {
		if SpecMatchesPublished(userSpec, v) {
			hits = append(hits, v)
		}
	}
	if len(hits) == 0 {
		return "", false
	}
	if len(hits) == 1 {
		return hits[0], true
	}
	sort.SliceStable(hits, func(i, j int) bool {
		ri := revisionSortKey(hits[i])
		rj := revisionSortKey(hits[j])
		if ri != rj {
			return ri > rj
		}
		return hits[i] > hits[j]
	})
	return hits[0], true
}

func revisionSortKey(v string) int {
	_, rev, ok := SplitBaseRevision(v)
	if !ok {
		return 0
	}
	n, err := strconv.Atoi(rev)
	if err != nil {
		return 0
	}
	return n
}
