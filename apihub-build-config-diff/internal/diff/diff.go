package diff

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

type BuildConfig struct {
	Refs []Ref `json:"refs"`
}

type Ref struct {
	ParentRefID   string `json:"parentRefId"`
	ParentVersion string `json:"parentVersion"`
	RefID         string `json:"refId"`
	Version       string `json:"version"`
}

type Summary struct {
	Added   int `json:"added"`
	Removed int `json:"removed"`
	Changed int `json:"changed"`
}

type FieldChange struct {
	Field string `json:"field"`
	Old   string `json:"old"`
	New   string `json:"new"`
}

type ChangedRef struct {
	RefID   string        `json:"refId"`
	Old     Ref           `json:"old"`
	New     Ref           `json:"new"`
	Changes []FieldChange `json:"changes"`
}

type Result struct {
	Summary Summary      `json:"summary"`
	Added   []Ref        `json:"added"`
	Removed []Ref        `json:"removed"`
	Changed []ChangedRef `json:"changed"`
}

type Filter struct {
	Added   bool
	Removed bool
	Changed bool
}

func ParseBuildConfig(r io.Reader) (BuildConfig, error) {
	var cfg BuildConfig
	if err := json.NewDecoder(r).Decode(&cfg); err != nil {
		return BuildConfig{}, err
	}
	if cfg.Refs == nil {
		return BuildConfig{}, fmt.Errorf("refs array is missing")
	}
	return cfg, nil
}

func Compare(oldRefs, newRefs []Ref) (Result, error) {
	oldByRefID, err := indexRefs(oldRefs, "old")
	if err != nil {
		return Result{}, err
	}
	newByRefID, err := indexRefs(newRefs, "new")
	if err != nil {
		return Result{}, err
	}

	var result Result
	for refID, oldRef := range oldByRefID {
		newRef, exists := newByRefID[refID]
		if !exists {
			result.Removed = append(result.Removed, oldRef)
			continue
		}
		if changes := compareRefFields(oldRef, newRef); len(changes) > 0 {
			result.Changed = append(result.Changed, ChangedRef{
				RefID:   refID,
				Old:     oldRef,
				New:     newRef,
				Changes: changes,
			})
		}
	}
	for refID, newRef := range newByRefID {
		if _, exists := oldByRefID[refID]; !exists {
			result.Added = append(result.Added, newRef)
		}
	}

	sortRefs(result.Added)
	sortRefs(result.Removed)
	sort.Slice(result.Changed, func(i, j int) bool {
		return result.Changed[i].RefID < result.Changed[j].RefID
	})

	result.Summary = Summary{
		Added:   len(result.Added),
		Removed: len(result.Removed),
		Changed: len(result.Changed),
	}
	return result, nil
}

func ParseFilter(value string) (Filter, error) {
	filter := Filter{Added: true, Removed: true, Changed: true}
	if strings.TrimSpace(value) == "" {
		return filter, nil
	}

	filter = Filter{}
	for _, part := range strings.Split(value, ",") {
		part = strings.ToLower(strings.TrimSpace(part))
		switch part {
		case "":
			continue
		case "added":
			filter.Added = true
		case "removed":
			filter.Removed = true
		case "changed":
			filter.Changed = true
		default:
			return Filter{}, fmt.Errorf("unsupported --only value %q, expected added, removed, changed", part)
		}
	}
	if !filter.Added && !filter.Removed && !filter.Changed {
		return Filter{}, fmt.Errorf("--only must include at least one category")
	}
	return filter, nil
}

func ApplyFilter(result Result, filter Filter) Result {
	if !filter.Added {
		result.Added = nil
	}
	if !filter.Removed {
		result.Removed = nil
	}
	if !filter.Changed {
		result.Changed = nil
	}
	result.Summary = Summary{
		Added:   len(result.Added),
		Removed: len(result.Removed),
		Changed: len(result.Changed),
	}
	return result
}

func indexRefs(refs []Ref, label string) (map[string]Ref, error) {
	index := make(map[string]Ref, len(refs))
	for i, ref := range refs {
		ref.RefID = strings.TrimSpace(ref.RefID)
		if ref.RefID == "" {
			return nil, fmt.Errorf("%s refs[%d] has empty refId", label, i)
		}
		if _, exists := index[ref.RefID]; exists {
			return nil, fmt.Errorf("%s refs contain duplicate refId %q", label, ref.RefID)
		}
		index[ref.RefID] = ref
	}
	return index, nil
}

func compareRefFields(oldRef, newRef Ref) []FieldChange {
	var changes []FieldChange
	addChange := func(field, oldValue, newValue string) {
		if oldValue != newValue {
			changes = append(changes, FieldChange{Field: field, Old: oldValue, New: newValue})
		}
	}
	addChange("version", oldRef.Version, newRef.Version)
	addChange("parentRefId", oldRef.ParentRefID, newRef.ParentRefID)
	addChange("parentVersion", oldRef.ParentVersion, newRef.ParentVersion)
	return changes
}

func sortRefs(refs []Ref) {
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].RefID == refs[j].RefID {
			return refs[i].Version < refs[j].Version
		}
		return refs[i].RefID < refs[j].RefID
	})
}
