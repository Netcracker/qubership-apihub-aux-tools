package merge

import (
	"fmt"
	"os"
	"sync"
	"testing"

	"apihub-ddl-import/internal/ddl"
	"apihub-ddl-import/internal/model"
	"apihub-ddl-import/internal/xlsxin"
)

var (
	ddlOnce  sync.Once
	ddlModel *model.DDLModel
	ddlErr   error
)

func fixtureDDL(t *testing.T) *model.DDLModel {
	t.Helper()
	ddlOnce.Do(func() {
		data, err := os.ReadFile("../../testdata/ddl/base.sql")
		if err != nil {
			ddlErr = err
			return
		}
		ddlModel, ddlErr = ddl.ParseFiles([]model.File{{RelPath: "ddl/base.sql", Data: data}})
	})
	if ddlErr != nil {
		t.Fatalf("fixture DDL: %v", ddlErr)
	}
	return ddlModel
}

func mergeCase(t *testing.T, name string) *model.MergeResult {
	t.Helper()
	data, err := os.ReadFile(fmt.Sprintf("../../testdata/cases/%s/comments-and-pfk.xlsx", name))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := xlsxin.Read(data)
	if err != nil {
		t.Fatalf("%s: read workbook: %v", name, err)
	}
	return Merge(fixtureDDL(t), doc)
}

func countByCode(res *model.MergeResult) map[string]int {
	m := map[string]int{}
	for _, w := range res.Warnings {
		m[w.Code]++
	}
	return m
}

func dumpWarnings(t *testing.T, res *model.MergeResult, limit int) {
	t.Helper()
	for i, w := range res.Warnings {
		if i >= limit {
			t.Logf("... and %d more warnings", len(res.Warnings)-limit)
			break
		}
		t.Logf("  [%s] %s", w.Code, w.Msg)
	}
}

func TestCase01FullMatch(t *testing.T) {
	res := mergeCase(t, "case-01")
	if len(res.Warnings) != 0 {
		dumpWarnings(t, res, 40)
		t.Fatalf("warnings = %d, want 0", len(res.Warnings))
	}
	if len(res.TableComments) != 75 {
		t.Errorf("table comments = %d, want 75", len(res.TableComments))
	}
	colComments := 0
	for _, m := range res.ColumnComments {
		colComments += len(m)
	}
	if colComments != 1081 {
		t.Errorf("column comments = %d, want 1081", colComments)
	}
	if res.Stats.FKsGenerated != 38 || len(res.FKs) != 38 {
		for _, fk := range res.FKs {
			t.Logf("  FK %s.%s -> %s.%s", fk.Table, fk.Column, fk.TargetTable, fk.TargetColumn)
		}
		t.Errorf("FKs generated = %d (stats %d), want 38", len(res.FKs), res.Stats.FKsGenerated)
	}
	if res.Stats.FKsUnresolved != 0 || res.Stats.FKsAmbiguous != 0 {
		t.Errorf("unresolved=%d ambiguous=%d, want 0/0", res.Stats.FKsUnresolved, res.Stats.FKsAmbiguous)
	}
	if len(res.Domains) != 12 {
		t.Errorf("domains = %v, want 12 domains", res.Domains)
	}
	if res.Stats.TablesMatched != 75 || res.Stats.ColumnsMatched != 1081 {
		t.Errorf("matched tables/columns = %d/%d, want 75/1081", res.Stats.TablesMatched, res.Stats.ColumnsMatched)
	}
}

func TestCase02MissingEntities(t *testing.T) {
	res := mergeCase(t, "case-02")
	c := countByCode(res)
	if c[model.WTblDdlOnly] != 3 {
		t.Errorf("TBL_DDL_ONLY = %d, want 3", c[model.WTblDdlOnly])
	}
	if c[model.WTblXlsxOnly] != 2 {
		t.Errorf("TBL_XLSX_ONLY = %d, want 2", c[model.WTblXlsxOnly])
	}
	if c[model.WColDdlOnly] != 50 {
		t.Errorf("COL_DDL_ONLY = %d, want 50 (columns of dropped tables must be counted)", c[model.WColDdlOnly])
	}
	if c[model.WColXlsxOnly] != 12 {
		t.Errorf("COL_XLSX_ONLY = %d, want 12", c[model.WColXlsxOnly])
	}
	for code, n := range c {
		switch code {
		case model.WTblDdlOnly, model.WTblXlsxOnly, model.WColDdlOnly, model.WColXlsxOnly:
		default:
			t.Errorf("unexpected warning category %s (%d)", code, n)
		}
	}
}

func TestCase03MissingDescriptions(t *testing.T) {
	res := mergeCase(t, "case-03")
	c := countByCode(res)
	if c[model.WDescEmptyTable] != 5 {
		t.Errorf("DESC_EMPTY_TABLE = %d, want 5", c[model.WDescEmptyTable])
	}
	if c[model.WDescEmptyColumn] != 40 {
		t.Errorf("DESC_EMPTY_COLUMN = %d, want 40", c[model.WDescEmptyColumn])
	}
	if res.Stats.EmptyTableDesc != 5 || res.Stats.EmptyColumnDesc != 40 {
		t.Errorf("stats empty desc = %d/%d, want 5/40", res.Stats.EmptyTableDesc, res.Stats.EmptyColumnDesc)
	}
	if len(res.TableComments) != 70 {
		t.Errorf("table comments = %d, want 70", len(res.TableComments))
	}
}

func TestCase04NameCaseMismatch(t *testing.T) {
	res := mergeCase(t, "case-04")
	if len(res.Warnings) != 0 {
		dumpWarnings(t, res, 40)
		t.Fatalf("warnings = %d, want 0 (matching must be case- and separator-insensitive)", len(res.Warnings))
	}
	if res.Stats.TablesMatched != 75 || res.Stats.ColumnsMatched != 1081 {
		t.Errorf("matched = %d/%d, want 75/1081", res.Stats.TablesMatched, res.Stats.ColumnsMatched)
	}
}

func TestCase05PkConflict(t *testing.T) {
	res := mergeCase(t, "case-05")
	c := countByCode(res)
	if c[model.WPkMismatch] != 10 || res.Stats.PKMismatchTables != 10 {
		dumpWarnings(t, res, 40)
		t.Errorf("PK_MISMATCH = %d (stats %d), want 10", c[model.WPkMismatch], res.Stats.PKMismatchTables)
	}
	for code, n := range c {
		if code != model.WPkMismatch {
			t.Errorf("unexpected warning category %s (%d)", code, n)
		}
	}
}

func TestCase06FkUnresolvable(t *testing.T) {
	res := mergeCase(t, "case-06")
	c := countByCode(res)
	if res.Stats.FKsGenerated != 38 {
		t.Errorf("FKs generated = %d, want 38", res.Stats.FKsGenerated)
	}
	if got := res.Stats.FKsUnresolved + res.Stats.FKsAmbiguous; got != 11 {
		dumpWarnings(t, res, 60)
		t.Errorf("FK unresolved+ambiguous = %d (%d+%d), want 11",
			got, res.Stats.FKsUnresolved, res.Stats.FKsAmbiguous)
	}
	if c[model.WPfkWithoutPk] != 3 {
		t.Errorf("PFK_WITHOUT_PK = %d, want 3", c[model.WPfkWithoutPk])
	}

	targets := map[string]string{}
	for _, fk := range res.FKs {
		targets[fk.Table+"."+fk.Column] = fk.TargetTable
	}
	spot := map[string]string{
		"slm_granite_timed_oaken.timed_id": "slm_granite_timed",
		"pmwp_chronicle_opal.chronicle_id": "ees_chronicle",
	}
	for from, want := range spot {
		if got, ok := targets[from]; ok && got != want {
			t.Errorf("FK %s -> %s, want %s", from, got, want)
		}
	}
	for _, fk := range res.FKs {
		if fk.Column == "sable_thicket_id" && fk.TargetTable != "bc_sable_thicket" {
			t.Errorf("FK %s.sable_thicket_id -> %s, want bc_sable_thicket", fk.Table, fk.TargetTable)
		}
	}
}

func TestCase07DirtyValues(t *testing.T) {
	res := mergeCase(t, "case-07")
	c := countByCode(res)
	if res.Stats.TablesXlsx != 77 || res.Stats.ColumnsXlsx != 1094 {
		t.Errorf("raw xlsx rows = %d/%d, want 77/1094", res.Stats.TablesXlsx, res.Stats.ColumnsXlsx)
	}
	if res.Stats.DupExactRows != 12 {
		dumpWarnings(t, res, 60)
		t.Errorf("DUP_EXACT rows = %d, want 12 (10 column + 2 table rows)", res.Stats.DupExactRows)
	}
	if res.Stats.DupConflicts != 3 || c[model.WDupConflict] != 3 {
		dumpWarnings(t, res, 60)
		t.Errorf("DUP_CONFLICT = %d (stats %d), want 3", c[model.WDupConflict], res.Stats.DupConflicts)
	}
	if c[model.WBadFlag] != 0 {
		t.Errorf("BAD_FLAG = %d, want 0 (YES/1/true/pk must canonicalize)", c[model.WBadFlag])
	}
	// The three conflicting descriptions must not be silently picked: total
	// column comments = matched columns minus the 3 conflicted ones.
	colComments := 0
	for _, m := range res.ColumnComments {
		colComments += len(m)
	}
	if colComments != res.Stats.ColumnsMatched-3 {
		t.Errorf("column comments = %d, matched = %d — conflicted descriptions must be skipped",
			colComments, res.Stats.ColumnsMatched)
	}
}

func TestCase10CombinedRealistic(t *testing.T) {
	res := mergeCase(t, "case-10")
	c := countByCode(res)
	if c[model.WTblDdlOnly] != 2 {
		t.Errorf("TBL_DDL_ONLY = %d, want 2", c[model.WTblDdlOnly])
	}
	if c[model.WColXlsxOnly] != 3 {
		t.Errorf("COL_XLSX_ONLY = %d, want 3", c[model.WColXlsxOnly])
	}
	if c[model.WDescEmptyTable] != 3 {
		t.Errorf("DESC_EMPTY_TABLE = %d, want 3", c[model.WDescEmptyTable])
	}
	if c[model.WDescEmptyColumn] != 25 {
		t.Errorf("DESC_EMPTY_COLUMN = %d, want 25", c[model.WDescEmptyColumn])
	}
	// Columns of the two workbook-missing tables must all be counted.
	ddlm := fixtureDDL(t)
	wantColOnly := 0
	for _, tb := range ddlm.Tables {
		if tb.Name == "acm_diploma_dormer_parapet_harbor_v4" || tb.Name == "ees_chronicle" {
			wantColOnly += len(tb.Columns)
		}
	}
	if c[model.WColDdlOnly] != wantColOnly {
		t.Errorf("COL_DDL_ONLY = %d, want %d (all columns of the two dropped tables)", c[model.WColDdlOnly], wantColOnly)
	}
}

func TestCanonFlags(t *testing.T) {
	for in, want := range map[string]string{"Y": "Y", "YES": "Y", "Yes": "Y", "y": "Y", "1": "Y", "true": "Y", "TRUE": "Y", "": "", "N": "", "no": "", "0": "", "false": ""} {
		got, ok := CanonIsPK(in)
		if !ok || got != want {
			t.Errorf("CanonIsPK(%q) = %q ok=%v, want %q", in, got, ok, want)
		}
	}
	if _, ok := CanonIsPK("maybe"); ok {
		t.Error("CanonIsPK(maybe) must not be ok")
	}
	for in, want := range map[string]string{"pk": "PK", "Pk": "PK", "pK": "PK", "PK": "PK", "fk": "FK", "PFK": "PFK", "pfk": "PFK", "": ""} {
		got, ok := CanonConstraint(in)
		if !ok || got != want {
			t.Errorf("CanonConstraint(%q) = %q ok=%v, want %q", in, got, ok, want)
		}
	}
	if _, ok := CanonConstraint("XX"); ok {
		t.Error("CanonConstraint(XX) must not be ok")
	}
}
