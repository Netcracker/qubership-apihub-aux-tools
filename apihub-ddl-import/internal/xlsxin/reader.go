// Package xlsxin reads the comments workbook (sheets "List of Tables" and
// "Tables Specifications") and validates its schema before any merge work.
package xlsxin

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"

	"apihub-ddl-import/internal/model"
)

// Expected sheet names and header rows (the workbook contract).
const (
	SheetTables = "List of Tables"
	SheetSpecs  = "Tables Specifications"
)

var (
	tablesHeaders = []string{"Domain", "Table Name", "Table Description", "External Table"}
	specsHeaders  = []string{"Domain", "Table Name", "Column Name", "Data Type", "IsPK",
		"Constraint", "Column Description", "Deployment Release", "RDB", "ExternalDB"}
)

// Read parses the workbook bytes. Schema violations (missing sheets, renamed or
// missing headers) and a fully empty workbook are fatal; a single empty sheet is
// only a warning.
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

	tableRows, err := readSheet(wb, tablesSheet, tablesHeaders)
	if err != nil {
		return nil, err
	}
	specRows, err := readSheet(wb, specsSheet, specsHeaders)
	if err != nil {
		return nil, err
	}

	doc := &model.CommentsDoc{}
	for _, r := range tableRows {
		doc.Tables = append(doc.Tables, model.TableRow{
			Domain:   r.cells[0],
			Table:    r.cells[1],
			Desc:     r.cells[2],
			External: r.cells[3],
			Row:      r.num,
		})
	}
	for _, r := range specRows {
		doc.Columns = append(doc.Columns, model.ColumnRow{
			Domain:        r.cells[0],
			Table:         r.cells[1],
			Column:        r.cells[2],
			DataType:      r.cells[3],
			IsPK:          r.cells[4],
			Constraint:    r.cells[5],
			Desc:          r.cells[6],
			DeployRelease: r.cells[7],
			RDB:           r.cells[8],
			ExternalDB:    r.cells[9],
			Row:           r.num,
		})
	}

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
	num   int // 1-based row number in the sheet
	cells []string
}

// readSheet validates the header row and returns non-empty data rows with cells
// trimmed and padded to the expected width.
func readSheet(wb *excelize.File, sheet string, headers []string) ([]sheetRow, error) {
	rows, err := wb.GetRows(sheet)
	if err != nil {
		return nil, model.Fatalf(model.FXlsxHeader, "read sheet %q: %v", sheet, err)
	}
	if len(rows) == 0 {
		return nil, model.Fatalf(model.FXlsxHeader, "sheet %q is empty (no header row)", sheet)
	}
	if err := validateHeader(sheet, rows[0], headers); err != nil {
		return nil, err
	}
	var out []sheetRow
	for i := 1; i < len(rows); i++ {
		cells := make([]string, len(headers))
		empty := true
		for j := range headers {
			v := ""
			if j < len(rows[i]) {
				v = strings.TrimSpace(rows[i][j])
			}
			cells[j] = v
			if v != "" {
				empty = false
			}
		}
		if empty {
			continue
		}
		out = append(out, sheetRow{num: i + 1, cells: cells})
	}
	return out, nil
}

func validateHeader(sheet string, got []string, want []string) error {
	var problems []string
	for i, w := range want {
		g := ""
		if i < len(got) {
			g = strings.TrimSpace(got[i])
		}
		if !strings.EqualFold(g, w) {
			if g == "" {
				problems = append(problems, fmt.Sprintf("column %s: missing header %q", colName(i), w))
			} else {
				problems = append(problems, fmt.Sprintf("column %s: got %q, want %q", colName(i), g, w))
			}
		}
	}
	if len(problems) > 0 {
		return model.Fatalf(model.FXlsxHeader, "sheet %q header mismatch: %s", sheet, strings.Join(problems, "; "))
	}
	return nil
}

func colName(idx int) string {
	n, _ := excelize.ColumnNumberToName(idx + 1)
	return n
}
