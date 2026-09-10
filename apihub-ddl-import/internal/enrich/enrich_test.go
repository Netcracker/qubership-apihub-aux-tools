package enrich

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/xuri/excelize/v2"

	"apihub-ddl-import/internal/model"
)

func buildWorkbook(t *testing.T, sheet string, headers []string, rows [][]any) []byte {
	t.Helper()
	f := excelize.NewFile()
	if _, err := f.NewSheet(sheet); err != nil {
		t.Fatal(err)
	}
	if err := f.DeleteSheet("Sheet1"); err != nil && sheet != "Sheet1" {
		t.Fatal(err)
	}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := f.SetCellValue(sheet, cell, h); err != nil {
			t.Fatal(err)
		}
	}
	for r, row := range rows {
		for c, v := range row {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+2)
			if err := f.SetCellValue(sheet, cell, v); err != nil {
				t.Fatal(err)
			}
		}
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	return buf.Bytes()
}

func readSheet(t *testing.T, data []byte) [][]string {
	t.Helper()
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := f.GetRows(DdlSheet)
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

var entitiesHeadersCurrent = []string{"Package ID", "Package Name", "Version", "Schema Name", "Name", "Description", "Document ID", "Group"}
var entitiesHeadersLegacy = entitiesHeadersCurrent[:7]

func TestEntitiesFillsOnlyEmptyGroupCells(t *testing.T) {
	data := buildWorkbook(t, DdlSheet, entitiesHeadersCurrent, [][]any{
		{"p", "P", "v1", "public", "acm_first", "d", "doc1", "ACM"},
		{"p", "P", "v1", "public", "acm_second", "d", "doc2", ""},
		{"p", "P", "v1", "public", "acm_third", "d", "doc3", "BC, QM"},
	})
	opts := Options{DomainByTable: map[string]string{
		model.NormKey("acm_first"):  "ACM",
		model.NormKey("acm_second"): "ACM",
		model.NormKey("acm_third"):  "ACM",
	}}
	out, info, err := Entities(data, opts)
	if err != nil {
		t.Fatal(err)
	}
	if info.GroupAppended {
		t.Error("Group column exists, must not be appended")
	}
	if info.GroupFilled != 1 {
		t.Errorf("GroupFilled = %d, want 1", info.GroupFilled)
	}
	if n := len(info.Warnings); n != 1 || info.Warnings[0].Code != model.WExportGroupMismatch {
		t.Errorf("warnings = %+v, want one EXPORT_GROUP_MISMATCH", info.Warnings)
	}
	rows := readSheet(t, out)
	if rows[2][7] != "ACM" {
		t.Errorf("empty Group cell not filled: %v", rows[2])
	}
	if rows[3][7] != "BC, QM" {
		t.Errorf("backend Group value must be kept: %v", rows[3])
	}
}

func TestEntitiesAppendsGroupOnLegacyLayout(t *testing.T) {
	data := buildWorkbook(t, DdlSheet, entitiesHeadersLegacy, [][]any{
		{"p", "P", "v1", "public", "bc_table", "d", "doc1"},
	})
	out, info, err := Entities(data, Options{DomainByTable: map[string]string{model.NormKey("bc_table"): "BC"}})
	if err != nil {
		t.Fatal(err)
	}
	if !info.GroupAppended || info.GroupFilled != 1 {
		t.Errorf("appended=%v filled=%d, want true/1", info.GroupAppended, info.GroupFilled)
	}
	rows := readSheet(t, out)
	if rows[0][7] != "Group" {
		t.Errorf("header row = %v, want Group at H1", rows[0])
	}
	if rows[1][7] != "BC" {
		t.Errorf("data row = %v", rows[1])
	}
}

var changesHeadersCounts = []string{"Version", "Previous Version", "Schema Name", "Name", "Kind",
	"Breaking", "Semi-breaking", "Deprecated", "Non-breaking", "Annotation", "Unclassified", "Group"}

func TestChangesCountLayoutWithPerChangeAPI(t *testing.T) {
	data := buildWorkbook(t, DdlSheet, changesHeadersCounts, [][]any{
		{"v2", "v1", "public", "acm_typed", "table", 0, 0, 0, 2, 0, 0, ""},
		{"v2", "v1", "public", "acm_removed", "table", 1, 0, 0, 0, 0, 0, ""},
		{"v2", "v1", "public", "acm_annot", "table", 0, 0, 0, 0, 1, 0, ""},
		{"v2", "v1", "public", "acm_plain", "table", 0, 0, 0, 3, 0, 0, ""},
	})
	perChange := map[string][]Change{
		"tbl-typed": {{Description: "Column 'x' type changed from text to int4", Severity: "non-breaking"}},
		"tbl-annot": {{Description: "Comment updated", Severity: "annotation"}},
		"tbl-plain": {{Description: "Column added", Severity: "non-breaking"}},
	}
	opts := Options{
		DomainByTable: map[string]string{
			model.NormKey("acm_typed"): "ACM", model.NormKey("acm_annot"): "ACM", model.NormKey("acm_plain"): "ACM",
		},
		EntityIDByName: map[string]string{
			model.NormKey("acm_typed"): "tbl-typed", model.NormKey("acm_annot"): "tbl-annot", model.NormKey("acm_plain"): "tbl-plain",
		},
		FetchChanges: func(id string) ([]Change, error) {
			if chs, ok := perChange[id]; ok {
				return chs, nil
			}
			return nil, fmt.Errorf("no such entity %s", id)
		},
	}
	out, info, err := Changes(data, opts)
	if err != nil {
		t.Fatal(err)
	}
	if !info.AnalyticsFallback {
		t.Error("removed entity must trigger the counts fallback flag")
	}
	rows := readSheet(t, out)
	if rows[0][12] != "Analytics Severity" {
		t.Fatalf("header = %v, want Analytics Severity at M1", rows[0])
	}
	want := map[string]string{
		"acm_typed":   "breaking", // data-type exception beats non-breaking severity
		"acm_removed": "breaking", // fallback from Breaking count
		"acm_annot":   "breaking", // annotation → breaking
		"acm_plain":   "non-breaking",
	}
	for _, row := range rows[1:] {
		if got := row[12]; got != want[row[3]] {
			t.Errorf("%s: analytics = %q, want %q", row[3], got, want[row[3]])
		}
	}
	// Group filled from the domain map for rows with empty Group.
	if info.GroupFilled != 3 {
		t.Errorf("GroupFilled = %d, want 3 (acm_removed has no domain)", info.GroupFilled)
	}
}

func TestChangesPerChangeLayout(t *testing.T) {
	headers := []string{"Package ID", "Package Name", "Version", "Previous Version", "Group",
		"Schema Name", "Table Name", "Name", "Change Description", "Change Severity", "Kind"}
	data := buildWorkbook(t, DdlSheet, headers, [][]any{
		{"p", "P", "v2", "v1", "", "public", "t1", "t1", "[Added] table 't1'", "non-breaking", "table"},
		{"p", "P", "v2", "v1", "", "public", "t2", "t2", "Column 'a' Data Type changed", "non-breaking", "table"},
		{"p", "P", "v2", "v1", "", "public", "t3", "t3", "something odd", "requiring attention", "table"},
		{"p", "P", "v2", "v1", "", "public", "t4", "t4", "??", "unclassified", "table"},
	})
	out, info, err := Changes(data, Options{DomainByTable: map[string]string{model.NormKey("t1"): "ACM"}})
	if err != nil {
		t.Fatal(err)
	}
	if info.AnalyticsFallback {
		t.Error("per-change layout must not use the counts fallback")
	}
	rows := readSheet(t, out)
	analyticsIdx := len(headers) // appended after the last header
	if rows[0][analyticsIdx] != "Analytics Severity" {
		t.Fatalf("header = %v", rows[0])
	}
	want := []string{"non-breaking", "breaking", "breaking", "unclassified"}
	for i, w := range want {
		if rows[i+1][analyticsIdx] != w {
			t.Errorf("row %d analytics = %q, want %q", i+1, rows[i+1][analyticsIdx], w)
		}
	}
}

func TestUnknownLayoutKeptAsIs(t *testing.T) {
	data := buildWorkbook(t, "Data", []string{"Whatever"}, [][]any{{"x"}})
	out, info, err := Entities(data, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, data) {
		t.Error("unknown layout must be returned unmodified")
	}
	if len(info.Warnings) != 1 || info.Warnings[0].Code != model.WExportLayoutUnknown {
		t.Errorf("warnings = %+v", info.Warnings)
	}
}

func TestMapAnalytics(t *testing.T) {
	re := DefaultDataTypeRegex
	cases := []struct {
		sev, desc, want string
	}{
		{"breaking", "", "breaking"},
		{"semi-breaking", "", "breaking"},
		{"requiring attention", "", "breaking"},
		{"annotation", "", "breaking"},
		{"deprecated", "", "breaking"},
		{"non-breaking", "", "non-breaking"},
		{"unclassified", "", "unclassified"},
		{"weird", "", "unclassified"},
		{"non-breaking", "Column 'x' type changed from a to b", "breaking"},
		{"unclassified", "Data type updated", "breaking"},
		{"non-breaking", "Changed the data type of column x", "breaking"},
		{"non-breaking", "Column comment updated", "non-breaking"},
	}
	for _, c := range cases {
		if got := mapAnalytics(c.sev, c.desc, re); got != c.want {
			t.Errorf("mapAnalytics(%q, %q) = %q, want %q", c.sev, c.desc, got, c.want)
		}
	}
}
