package diff

import (
	"strings"
	"testing"
)

func TestCompareReportsAddedRemovedAndChangedRefs(t *testing.T) {
	oldRefs := []Ref{
		{RefID: "unchanged", Version: "1.0.0", ParentRefID: "app-1", ParentVersion: "app-version-1"},
		{RefID: "changed", Version: "1.0.0", ParentRefID: "app-1", ParentVersion: "app-version-1"},
		{RefID: "removed", Version: "1.0.0"},
	}
	newRefs := []Ref{
		{RefID: "unchanged", Version: "1.0.0", ParentRefID: "app-1", ParentVersion: "app-version-1"},
		{RefID: "changed", Version: "2.0.0", ParentRefID: "app-2", ParentVersion: "app-version-2"},
		{RefID: "added", Version: "1.0.0"},
	}

	result, err := Compare(oldRefs, newRefs)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	if result.Summary != (Summary{Added: 1, Removed: 1, Changed: 1}) {
		t.Fatalf("unexpected summary: %+v", result.Summary)
	}
	if got := result.Added[0].RefID; got != "added" {
		t.Fatalf("added refId = %q, want added", got)
	}
	if got := result.Removed[0].RefID; got != "removed" {
		t.Fatalf("removed refId = %q, want removed", got)
	}
	if got := result.Changed[0].RefID; got != "changed" {
		t.Fatalf("changed refId = %q, want changed", got)
	}

	wantChanges := []FieldChange{
		{Field: "version", Old: "1.0.0", New: "2.0.0"},
		{Field: "parentRefId", Old: "app-1", New: "app-2"},
		{Field: "parentVersion", Old: "app-version-1", New: "app-version-2"},
	}
	assertFieldChanges(t, result.Changed[0].Changes, wantChanges)
}

func TestCompareSortsOutputDeterministically(t *testing.T) {
	oldRefs := []Ref{
		{RefID: "z-removed", Version: "1"},
		{RefID: "b-changed", Version: "1"},
		{RefID: "a-changed", Version: "1"},
	}
	newRefs := []Ref{
		{RefID: "c-added", Version: "1"},
		{RefID: "a-added", Version: "1"},
		{RefID: "b-changed", Version: "2"},
		{RefID: "a-changed", Version: "2"},
	}

	result, err := Compare(oldRefs, newRefs)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	assertRefIDs(t, "added", refsToIDs(result.Added), []string{"a-added", "c-added"})
	assertRefIDs(t, "removed", refsToIDs(result.Removed), []string{"z-removed"})
	assertRefIDs(t, "changed", changedToIDs(result.Changed), []string{"a-changed", "b-changed"})
}

func TestCompareRejectsDuplicateRefID(t *testing.T) {
	_, err := Compare(
		[]Ref{{RefID: "duplicate"}, {RefID: "duplicate"}},
		[]Ref{},
	)
	if err == nil {
		t.Fatal("Compare() error = nil, want duplicate refId error")
	}
	if !strings.Contains(err.Error(), `duplicate refId "duplicate"`) {
		t.Fatalf("Compare() error = %q, want duplicate refId message", err)
	}
}

func TestParseBuildConfigRequiresRefsArray(t *testing.T) {
	_, err := ParseBuildConfig(strings.NewReader(`{"packageId":"example"}`))
	if err == nil {
		t.Fatal("ParseBuildConfig() error = nil, want missing refs error")
	}
	if !strings.Contains(err.Error(), "refs array is missing") {
		t.Fatalf("ParseBuildConfig() error = %q, want missing refs message", err)
	}
}

func TestParseFilter(t *testing.T) {
	filter, err := ParseFilter("added,removed")
	if err != nil {
		t.Fatalf("ParseFilter() error = %v", err)
	}
	if !filter.Added || !filter.Removed || filter.Changed {
		t.Fatalf("unexpected filter: %+v", filter)
	}

	_, err = ParseFilter("unknown")
	if err == nil {
		t.Fatal("ParseFilter() error = nil, want unsupported value error")
	}
}

func TestApplyFilterUpdatesVisibleSummary(t *testing.T) {
	result := Result{
		Summary: Summary{Added: 1, Removed: 1, Changed: 1},
		Added:   []Ref{{RefID: "added"}},
		Removed: []Ref{{RefID: "removed"}},
		Changed: []ChangedRef{{RefID: "changed"}},
	}

	filtered := ApplyFilter(result, Filter{Added: true, Removed: true})
	if filtered.Summary != (Summary{Added: 1, Removed: 1}) {
		t.Fatalf("unexpected filtered summary: %+v", filtered.Summary)
	}
	if len(filtered.Changed) != 0 {
		t.Fatalf("filtered changed length = %d, want 0", len(filtered.Changed))
	}
}

func assertFieldChanges(t *testing.T, got, want []FieldChange) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("changes length = %d, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("changes[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func assertRefIDs(t *testing.T, label string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s refId length = %d, want %d: %+v", label, len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s refIds[%d] = %q, want %q", label, i, got[i], want[i])
		}
	}
}

func refsToIDs(refs []Ref) []string {
	ids := make([]string, 0, len(refs))
	for _, ref := range refs {
		ids = append(ids, ref.RefID)
	}
	return ids
}

func changedToIDs(refs []ChangedRef) []string {
	ids := make([]string, 0, len(refs))
	for _, ref := range refs {
		ids = append(ids, ref.RefID)
	}
	return ids
}
