// Command gen-comments-v2 evolves the e2e-demo comments workbook to match the
// "next version" DDL (seed-v2/ddl/apihub_ddl.sql): it edits the v1 workbook
// incrementally (add/update/remove specific rows) rather than regenerating it,
// mirroring how a real comments doc would be maintained alongside a schema
// change. The simulated change set:
//
//	report_subscription   new table, domain "Reporting" (a brand-new group),
//	                      build_id left undeclared in the DDL + marked FK in
//	                      the workbook -> resolves cleanly to build(build_id)
//	build.retry_after     new nullable column on an existing table
//	endpoint_calls.count  int -> bigint (data-type change)
//	csv_dashboard_publication.message   column dropped (DDL and workbook row)
//
// Self-verifies against the tool's own parser+merge before writing, same as
// gen-comments: asserts full coverage and the exact expected warning/stat
// delta (unchanged 6 base warnings, +1 table, +8 columns, +1 generated FK).
//
// Usage: go run ./e2e-demo/gen-comments-v2 -ddl <v2 dir> -in <v1 xlsx> -out <v2 xlsx>
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xuri/excelize/v2"

	"apihub-ddl-import/internal/ddl"
	"apihub-ddl-import/internal/merge"
	"apihub-ddl-import/internal/model"
	"apihub-ddl-import/internal/xlsxin"
)

const (
	deployRelease  = "2026.5"
	rdbName        = "PostgreSQL"
	droppedTable   = "csv_dashboard_publication"
	droppedColumn  = "message"
	newTable       = "report_subscription"
	changedTable   = "endpoint_calls"
	changedColumn  = "count"
	newTableDomain = "Reporting"
)

func main() {
	ddlDir := flag.String("ddl", "", "v2 DDL directory")
	in := flag.String("in", "", "v1 workbook path")
	out := flag.String("out", "", "v2 workbook output path")
	flag.Parse()
	if *ddlDir == "" || *in == "" || *out == "" {
		fail("missing -ddl/-in/-out")
	}

	f, err := excelize.OpenFile(*in)
	if err != nil {
		fail("open v1 workbook: %v", err)
	}
	if err := f.SetDefaultFont("Arial"); err != nil {
		fail("set font: %v", err)
	}

	removeSpecRow(f, droppedTable, droppedColumn)
	updateDataType(f, changedTable, changedColumn, "bigint")
	appendTableRow(f, newTable, newTableDomain,
		"Per-user subscriptions to recurring report deliveries for a package, optionally pinned to a specific build.")
	appendSpecRows(f, newTable, newTableDomain, []specRow{
		{col: "id", typ: "varchar", isPK: "Y", cons: "PK", desc: "Primary identifier of the report subscription record."},
		{col: "package_id", typ: "varchar", desc: "Package the subscription's reports belong to."},
		{col: "user_id", typ: "varchar", desc: "User receiving the report deliveries."},
		{col: "build_id", typ: "varchar", cons: "FK", desc: "Build the subscription is pinned to; empty means \"latest\"."},
		{col: "frequency", typ: "varchar", desc: "Delivery cadence, e.g. daily or weekly."},
		{col: "created_at", typ: "timestamp", desc: "Timestamp when the subscription was created."},
		{col: "last_sent_at", typ: "timestamp", desc: "Timestamp of the last delivered report; empty before the first delivery."},
	})
	appendSpecRows(f, "build", "Builds", []specRow{
		{col: "retry_after", typ: "timestamp", desc: "Earliest time a failed build may be retried; empty when no retry is scheduled."},
	})

	if err := f.SaveAs(*out); err != nil {
		fail("save v2 workbook: %v", err)
	}
	f.Close()
	fmt.Printf("workbook written: %s\n\n", *out)

	verify(*ddlDir, *out)
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "gen-comments-v2: "+format+"\n", args...)
	os.Exit(1)
}

// findCol returns the 0-based index of a header cell, or -1.
func findCol(f *excelize.File, sheet, header string) int {
	rows, err := f.GetRows(sheet)
	if err != nil || len(rows) == 0 {
		return -1
	}
	for i, h := range rows[0] {
		if strings.EqualFold(strings.TrimSpace(h), header) {
			return i
		}
	}
	return -1
}

func removeSpecRow(f *excelize.File, table, column string) {
	tblIdx := findCol(f, xlsxin.SheetSpecs, "Table Name")
	colIdx := findCol(f, xlsxin.SheetSpecs, "Column Name")
	rows, err := f.GetRows(xlsxin.SheetSpecs)
	if err != nil {
		fail("read %s: %v", xlsxin.SheetSpecs, err)
	}
	for i := 1; i < len(rows); i++ {
		if rows[i][tblIdx] == table && rows[i][colIdx] == column {
			if err := f.RemoveRow(xlsxin.SheetSpecs, i+1); err != nil {
				fail("remove row for %s.%s: %v", table, column, err)
			}
			return
		}
	}
	fail("row for %s.%s not found, nothing to remove", table, column)
}

func updateDataType(f *excelize.File, table, column, newType string) {
	tblIdx := findCol(f, xlsxin.SheetSpecs, "Table Name")
	colIdx := findCol(f, xlsxin.SheetSpecs, "Column Name")
	typIdx := findCol(f, xlsxin.SheetSpecs, "Data Type")
	rows, err := f.GetRows(xlsxin.SheetSpecs)
	if err != nil {
		fail("read %s: %v", xlsxin.SheetSpecs, err)
	}
	for i := 1; i < len(rows); i++ {
		if rows[i][tblIdx] == table && rows[i][colIdx] == column {
			cell, _ := excelize.CoordinatesToCellName(typIdx+1, i+1)
			if err := f.SetCellValue(xlsxin.SheetSpecs, cell, newType); err != nil {
				fail("update data type for %s.%s: %v", table, column, err)
			}
			return
		}
	}
	fail("row for %s.%s not found, nothing to update", table, column)
}

func appendTableRow(f *excelize.File, table, domain, desc string) {
	rows, err := f.GetRows(xlsxin.SheetTables)
	if err != nil {
		fail("read %s: %v", xlsxin.SheetTables, err)
	}
	row := len(rows) + 1
	err = f.SetSheetRow(xlsxin.SheetTables, fmt.Sprintf("A%d", row), &[]any{domain, table, desc, "N"})
	if err != nil {
		fail("append table row for %s: %v", table, err)
	}
}

type specRow struct {
	col, typ, isPK, cons, desc string
}

func appendSpecRows(f *excelize.File, table, domain string, cols []specRow) {
	rows, err := f.GetRows(xlsxin.SheetSpecs)
	if err != nil {
		fail("read %s: %v", xlsxin.SheetSpecs, err)
	}
	row := len(rows) + 1
	for _, c := range cols {
		err := f.SetSheetRow(xlsxin.SheetSpecs, fmt.Sprintf("A%d", row),
			&[]any{domain, table, c.col, c.typ, c.isPK, c.cons, c.desc, deployRelease, rdbName, ""})
		if err != nil {
			fail("append spec row for %s.%s: %v", table, c.col, err)
		}
		row++
	}
}

// ---- self-verification, same shape as gen-comments/main.go ----

var expectedWarnings = map[string]int{
	model.WCommentExists: 3,
	model.WFkExists:      1,
	model.WFkUnresolved:  1,
	model.WFkAmbiguous:   1,
}

func verify(ddlDir, xlsxPath string) {
	files := loadDDLFiles(ddlDir)
	m, err := ddl.ParseFiles(files)
	if err != nil {
		fail("parse v2 DDL: %v", err)
	}
	data, err := os.ReadFile(xlsxPath)
	if err != nil {
		fail("re-read workbook: %v", err)
	}
	doc, err := xlsxin.Read(data)
	if err != nil {
		fail("workbook does not pass the tool's reader: %v", err)
	}
	res := merge.Merge(m, doc)
	st := res.Stats

	seen := map[string]bool{}
	var tables int
	var cols int
	for _, t := range m.Tables {
		k := model.NormKey(t.Name)
		if seen[k] {
			continue
		}
		seen[k] = true
		tables++
		cols += len(t.Columns)
	}

	fmt.Printf("merge self-check: %d/%d tables matched, %d/%d columns matched\n",
		st.TablesMatched, tables, st.ColumnsMatched, cols)
	fmt.Printf("FKs: %d generated, %d unresolved, %d ambiguous; domains: %d (%s)\n\n",
		st.FKsGenerated, st.FKsUnresolved, st.FKsAmbiguous, len(res.Domains), strings.Join(res.Domains, ", "))
	for _, fk := range res.FKs {
		if fk.Table == newTable {
			fmt.Printf("FK %s.%s -> %s(%s)\n", fk.Table, fk.Column, fk.TargetTable, fk.TargetColumn)
		}
	}
	byCode := map[string]int{}
	for _, w := range res.Warnings {
		byCode[w.Code]++
	}

	ok := true
	if st.TablesMatched != tables || st.ColumnsMatched != cols {
		ok = false
		fmt.Println("FAIL: incomplete coverage")
	}
	if tables != 73 {
		ok = false
		fmt.Printf("FAIL: expected 73 tables in v2 DDL, got %d\n", tables)
	}
	if cols != 499 {
		ok = false
		fmt.Printf("FAIL: expected 499 columns in v2 DDL, got %d\n", cols)
	}
	if st.FKsGenerated != 4 {
		ok = false
		fmt.Printf("FAIL: FKs generated: got %d, want 4\n", st.FKsGenerated)
	}
	for code, want := range expectedWarnings {
		if byCode[code] != want {
			ok = false
			fmt.Printf("FAIL: warning %s: got %d, want %d\n", code, byCode[code], want)
		}
	}
	for code, got := range byCode {
		if expectedWarnings[code] == 0 {
			ok = false
			fmt.Printf("FAIL: unexpected warning %s ×%d\n", code, got)
		}
	}
	found := false
	for _, d := range res.Domains {
		if d == newTableDomain {
			found = true
		}
	}
	if !found {
		ok = false
		fmt.Printf("FAIL: domain %q not present\n", newTableDomain)
	}
	if !ok {
		os.Exit(1)
	}
	fmt.Println("self-check PASSED: v2 workbook is clean against the v2 DDL")
}

func loadDDLFiles(dir string) []model.File {
	entries, err := os.ReadDir(dir)
	if err != nil {
		fail("read ddl dir: %v", err)
	}
	var paths []string
	for _, e := range entries {
		if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".sql") {
			paths = append(paths, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(paths)
	var files []model.File
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			fail("read %s: %v", p, err)
		}
		files = append(files, model.File{RelPath: filepath.Base(p), Data: data})
	}
	return files
}
