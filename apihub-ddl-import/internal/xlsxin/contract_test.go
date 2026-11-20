package xlsxin

import (
	"errors"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"

	"apihub-ddl-import/internal/model"
)

type testSheet struct {
	name string
	rows [][]string
}

// buildWorkbook creates an in-memory xlsx with the given sheets, in the given
// order, so tests can exercise header/sheet layouts without binary fixtures.
func buildWorkbook(t *testing.T, sheets []testSheet) []byte {
	t.Helper()
	f := excelize.NewFile()
	defer f.Close()

	for i, s := range sheets {
		name := s.name
		if i == 0 {
			if err := f.SetSheetName("Sheet1", name); err != nil {
				t.Fatal(err)
			}
		} else if _, err := f.NewSheet(name); err != nil {
			t.Fatal(err)
		}
		for r, row := range s.rows {
			for c, v := range row {
				cell, err := excelize.CoordinatesToCellName(c+1, r+1)
				if err != nil {
					t.Fatal(err)
				}
				if err := f.SetCellStr(name, cell, v); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// TestReadShuffledHeadersExtraSheetsAndColumns covers acceptance criteria #1
// and #2: four sheets in any order, required headers shuffled, plus extra
// columns, parses successfully and ignores sheets/columns outside the contract.
func TestReadShuffledHeadersExtraSheetsAndColumns(t *testing.T) {
	data := buildWorkbook(t, []testSheet{
		{name: "Version", rows: [][]string{{"garbage", "should never be read"}}},
		{name: "Thin Tables Specifications", rows: [][]string{{"also garbage"}}},
		{name: "Tables Specifications", rows: [][]string{
			{"Column Description", "Table Name", "Deployment Release", "Domain", "Column Name", "Constraint"},
			{"The id column", "bc_table_one", "2026.1", "DOMAIN_A", "table_one_id", "PK"},
		}},
		{name: "List of Tables", rows: [][]string{
			{"Deleted History", "Table Description", "Domain", "Table Name"},
			{"", "First table", "DOMAIN_A", "bc_table_one"},
		}},
	})

	doc, err := Read(data)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(doc.Tables) != 1 || doc.Tables[0].Table != "bc_table_one" || doc.Tables[0].Domain != "DOMAIN_A" {
		t.Errorf("Tables = %+v, want one row for bc_table_one/DOMAIN_A", doc.Tables)
	}
	if len(doc.Columns) != 1 || doc.Columns[0].Column != "table_one_id" || doc.Columns[0].Constraint != "PK" {
		t.Errorf("Columns = %+v, want one row for table_one_id/PK", doc.Columns)
	}
	if len(doc.Warnings) != 0 {
		t.Errorf("Warnings = %v, want none", doc.Warnings)
	}
}

// TestReadMissingRequiredHeaderFatal covers acceptance criteria #3: a missing
// required header is fatal and the error lists the missing header name.
func TestReadMissingRequiredHeaderFatal(t *testing.T) {
	data := buildWorkbook(t, []testSheet{
		{name: SheetTables, rows: [][]string{
			{"Domain", "Table Name", "Table Description", "External Table"},
			{"DOMAIN_A", "bc_table_one", "First table", ""},
		}},
		{name: SheetSpecs, rows: [][]string{
			{"Domain", "Table Name", "Column Name", "Data Type", "IsPK", "Constraint", "Deployment Release", "RDB", "ExternalDB"},
			{"DOMAIN_A", "bc_table_one", "table_one_id", "bigint", "Y", "PK", "2026.1", "", ""},
		}},
	})

	_, err := Read(data)
	var fe *model.FatalError
	if !errors.As(err, &fe) || fe.Code != model.FXlsxHeader {
		t.Fatalf("err = %v, want FatalError %s", err, model.FXlsxHeader)
	}
	if !strings.Contains(fe.Msg, "Column Description") {
		t.Errorf("fatal message %q does not name the missing header", fe.Msg)
	}
}

// TestReadMissingOptionalHeadersSuccess covers acceptance criteria #4: absent
// optional headers (Constraint, External Table, Data Type, ...) succeed, with
// the corresponding fields empty for every row.
func TestReadMissingOptionalHeadersSuccess(t *testing.T) {
	data := buildWorkbook(t, []testSheet{
		{name: SheetTables, rows: [][]string{
			{"Domain", "Table Name", "Table Description"},
			{"DOMAIN_A", "bc_table_one", "First table"},
		}},
		{name: SheetSpecs, rows: [][]string{
			{"Domain", "Table Name", "Column Name", "Column Description"},
			{"DOMAIN_A", "bc_table_one", "table_one_id", "The id column"},
		}},
	})

	doc, err := Read(data)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(doc.Tables) != 1 || doc.Tables[0].External != "" {
		t.Errorf("Tables = %+v, want one row with empty External", doc.Tables)
	}
	if len(doc.Columns) != 1 {
		t.Fatalf("Columns = %+v, want one row", doc.Columns)
	}
	c := doc.Columns[0]
	if c.DataType != "" || c.IsPK != "" || c.Constraint != "" || c.DeployRelease != "" || c.RDB != "" || c.ExternalDB != "" {
		t.Errorf("Columns[0] = %+v, want all optional fields empty", c)
	}
}

// TestReadRowMissingKeySkipped covers the "required row key" rule: a
// non-empty row missing Table Name (or Column Name) is skipped with a
// ROW_MISSING_KEY warning rather than failing the run or polluting the data.
func TestReadRowMissingKeySkipped(t *testing.T) {
	data := buildWorkbook(t, []testSheet{
		{name: SheetTables, rows: [][]string{
			{"Domain", "Table Name", "Table Description", "External Table"},
			{"DOMAIN_A", "", "Orphan description with no table name", ""},
			{"DOMAIN_A", "bc_table_one", "First table", ""},
		}},
		{name: SheetSpecs, rows: [][]string{
			{"Domain", "Table Name", "Column Name", "Column Description"},
			{"DOMAIN_A", "bc_table_one", "", "Orphan column description with no column name"},
			{"DOMAIN_A", "bc_table_one", "table_one_id", "The id column"},
		}},
	})

	doc, err := Read(data)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(doc.Tables) != 1 || doc.Tables[0].Table != "bc_table_one" {
		t.Errorf("Tables = %+v, want only the valid row", doc.Tables)
	}
	if len(doc.Columns) != 1 || doc.Columns[0].Column != "table_one_id" {
		t.Errorf("Columns = %+v, want only the valid row", doc.Columns)
	}
	n := model.CountWarnings(doc.Warnings, model.WRowMissingKey)
	if n != 2 {
		t.Errorf("ROW_MISSING_KEY warnings = %d, want 2; warnings: %v", n, doc.Warnings)
	}
}
