// Package xlsxin reads the comments workbook (sheets "List of Tables" and
// "Tables Specifications") and validates its schema before any merge work.
//
// Binding is name-based, per the workbook contract: sheets are looked up by
// name (any position in the file; any other sheet is ignored), and columns
// are looked up by header name (any position in the row; unknown/extra
// columns are ignored). Column order and physical position never matter.
package xlsxin

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"

	"apihub-ddl-import/internal/model"
)

// Expected sheet names (the workbook contract). Any other sheet (e.g.
// "Version", "Thin Tables Specifications") is ignored.
const (
	SheetTables = "List of Tables"
	SheetSpecs  = "Tables Specifications"
)

// column is one logical field read from a sheet by header name.
type column struct {
	header   string
	required bool // missing header is fatal
	rowKey   bool // non-empty data row with this field empty is skipped (warning)
}

var tablesColumns = []column{
	{header: "Domain", required: true},
	{header: "Table Name", required: true, rowKey: true},
	{header: "Table Description", required: true},
	{header: "External Table"},
}

var specsColumns = []column{
	{header: "Domain", required: true},
	{header: "Table Name", required: true, rowKey: true},
	{header: "Column Name", required: true, rowKey: true},
	{header: "Column Description", required: true},
	{header: "Data Type"},
	{header: "IsPK"},
	{header: "Constraint"},
	{header: "Deployment Release"},
	{header: "RDB"},
	{header: "ExternalDB"},
}

// Read parses the workbook bytes. Schema violations (missing sheets, missing
// required headers) and a fully empty workbook are fatal; a single empty
// sheet or a data row missing its required key is only a warning.
func Read(data []byte) (*model.CommentsDoc, error) {
	wb, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, model.Fatalf(model.FXlsxHeader, "cannot open workbook: %v", err)
	}
	defer wb.Close()

	tablesSheet, err := findSheet(wb, SheetTables)
	if err != nil {
		return nil, err
	}
	specsSheet, err := findSheet(wb, SheetSpecs)
	if err != nil {
		return nil, err
	}

	tableRows, tableWarnings, err := readSheet(wb, tablesSheet, SheetTables, tablesColumns)
	if err != nil {
		return nil, err
	}
	specRows, specWarnings, err := readSheet(wb, specsSheet, SheetSpecs, specsColumns)
	if err != nil {
		return nil, err
	}

	doc := &model.CommentsDoc{}
	for _, r := range tableRows {
		doc.Tables = append(doc.Tables, model.TableRow{
			Domain:   r.vals["Domain"],
			Table:    r.vals["Table Name"],
			Desc:     r.vals["Table Description"],
			External: r.vals["External Table"],
			Row:      r.num,
		})
	}
	for _, r := range specRows {
		doc.Columns = append(doc.Columns, model.ColumnRow{
			Domain:        r.vals["Domain"],
			Table:         r.vals["Table Name"],
			Column:        r.vals["Column Name"],
			DataType:      r.vals["Data Type"],
			IsPK:          r.vals["IsPK"],
			Constraint:    r.vals["Constraint"],
			Desc:          r.vals["Column Description"],
			DeployRelease: r.vals["Deployment Release"],
			RDB:           r.vals["RDB"],
			ExternalDB:    r.vals["ExternalDB"],
			Row:           r.num,
		})
	}
	doc.Warnings = append(doc.Warnings, tableWarnings...)
	doc.Warnings = append(doc.Warnings, specWarnings...)

	if len(doc.Tables) == 0 && len(doc.Columns) == 0 {
		return nil, model.Fatalf(model.FXlsxEmpty,
			"both sheets (%q, %q) contain no data rows — refusing to publish a version without any comments",
			SheetTables, SheetSpecs)
	}
	if len(doc.Tables) == 0 {
		doc.Warnings = append(doc.Warnings, model.Warning{
			Code: model.WSheetEmpty, Sheet: SheetTables,
			Msg: fmt.Sprintf("sheet %q has no data rows", SheetTables),
		})
	}
	if len(doc.Columns) == 0 {
		doc.Warnings = append(doc.Warnings, model.Warning{
			Code: model.WSheetEmpty, Sheet: SheetSpecs,
			Msg: fmt.Sprintf("sheet %q has no data rows", SheetSpecs),
		})
	}
	return doc, nil
}

func findSheet(wb *excelize.File, name string) (string, error) {
	for _, s := range wb.GetSheetList() {
		if strings.EqualFold(strings.TrimSpace(s), name) {
			return s, nil
		}
	}
	return "", model.Fatalf(model.FXlsxHeader, "sheet %q not found (sheets present: %s)",
		name, strings.Join(wb.GetSheetList(), ", "))
}

type sheetRow struct {
	num  int // 1-based row number in the sheet
	vals map[string]string
}

// headerIndex maps a column's trimmed, case-insensitive header name to its
// 0-based position in the header row. A repeated header keeps its first
// occurrence.
func headerIndex(headerRow []string) map[string]int {
	idx := make(map[string]int, len(headerRow))
	for i, h := range headerRow {
		key := strings.ToLower(strings.TrimSpace(h))
		if key == "" {
			continue
		}
		if _, exists := idx[key]; !exists {
			idx[key] = i
		}
	}
	return idx
}

// readSheet resolves cols by header name (order- and position-independent),
// validates required headers are present, and returns non-empty data rows.
// A non-empty row missing a required row-key field is dropped with a warning
// instead of failing the whole run.
func readSheet(wb *excelize.File, sheet, canonicalName string, cols []column) ([]sheetRow, []model.Warning, error) {
	rows, err := wb.GetRows(sheet)
	if err != nil {
		return nil, nil, model.Fatalf(model.FXlsxHeader, "read sheet %q: %v", canonicalName, err)
	}
	if len(rows) == 0 {
		return nil, nil, model.Fatalf(model.FXlsxHeader, "sheet %q is empty (no header row)", canonicalName)
	}

	idx := headerIndex(rows[0])
	colPos := make([]int, len(cols)) // -1 when the header is absent
	var missing []string
	for i, c := range cols {
		pos, ok := idx[strings.ToLower(c.header)]
		if !ok {
			colPos[i] = -1
			if c.required {
				missing = append(missing, c.header)
			}
			continue
		}
		colPos[i] = pos
	}
	if len(missing) > 0 {
		return nil, nil, model.Fatalf(model.FXlsxHeader,
			"sheet %q is missing required header(s): %s", canonicalName, strings.Join(missing, ", "))
	}

	var out []sheetRow
	var warnings []model.Warning
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		vals := make(map[string]string, len(cols))
		empty := true
		for ci, c := range cols {
			v := ""
			if p := colPos[ci]; p >= 0 && p < len(row) {
				v = strings.TrimSpace(row[p])
			}
			vals[c.header] = v
			if v != "" {
				empty = false
			}
		}
		if empty {
			continue
		}
		var missingKeys []string
		for _, c := range cols {
			if c.rowKey && vals[c.header] == "" {
				missingKeys = append(missingKeys, c.header)
			}
		}
		if len(missingKeys) > 0 {
			warnings = append(warnings, model.Warning{
				Code: model.WRowMissingKey, Sheet: canonicalName, Row: i + 1,
				Msg: fmt.Sprintf("row %d in %q skipped: required field(s) empty: %s",
					i+1, canonicalName, strings.Join(missingKeys, ", ")),
			})
			continue
		}
		out = append(out, sheetRow{num: i + 1, vals: vals})
	}
	return out, warnings, nil
}
