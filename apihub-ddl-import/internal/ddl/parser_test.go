package ddl

import (
	"errors"
	"os"
	"testing"

	"apihub-ddl-import/internal/model"
)

func loadFixtureModel(t *testing.T) *model.DDLModel {
	t.Helper()
	data, err := os.ReadFile("../../testdata/ddl/base.sql")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	m, err := ParseFiles([]model.File{{RelPath: "ddl/base.sql", Data: data}})
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	return m
}

func TestParseFixtureCounts(t *testing.T) {
	m := loadFixtureModel(t)
	if len(m.Files) != 1 {
		t.Fatalf("files = %d, want 1", len(m.Files))
	}
	if len(m.Tables) != 75 {
		t.Fatalf("tables = %d, want 75", len(m.Tables))
	}
	cols := 0
	withPK := 0
	composite := 0
	for _, tb := range m.Tables {
		cols += len(tb.Columns)
		if len(tb.PKCols) > 0 {
			withPK++
		}
		if len(tb.PKCols) > 1 {
			composite++
		}
		if tb.Schema != "public" {
			t.Errorf("table %s: schema = %q, want public", tb.Name, tb.Schema)
		}
	}
	if cols != 1081 {
		t.Errorf("columns = %d, want 1081", cols)
	}
	if withPK != 75 {
		t.Errorf("tables with PK = %d, want 75", withPK)
	}
	if composite != 11 {
		t.Errorf("tables with composite PK = %d, want 11", composite)
	}
}

func TestParseFixtureDetails(t *testing.T) {
	m := loadFixtureModel(t)
	byName := map[string]*model.DDLTable{}
	for _, tb := range m.Tables {
		byName[tb.Name] = tb
	}

	timed := byName["acm_timed_v4"]
	if timed == nil {
		t.Fatal("acm_timed_v4 not found")
	}
	wantPK := []string{"banner_approach_docket", "id", "spot_waiver_id"}
	if len(timed.PKCols) != len(wantPK) {
		t.Fatalf("acm_timed_v4 PK = %v, want %v", timed.PKCols, wantPK)
	}
	for i := range wantPK {
		if timed.PKCols[i] != wantPK[i] {
			t.Fatalf("acm_timed_v4 PK = %v, want %v", timed.PKCols, wantPK)
		}
	}

	sable := byName["bc_sable_thicket"]
	if sable == nil {
		t.Fatal("bc_sable_thicket not found")
	}
	if len(sable.PKCols) != 1 || sable.PKCols[0] != "sable_thicket_id" {
		t.Errorf("bc_sable_thicket PK = %v", sable.PKCols)
	}
	if sable.PKName != "bc_sable_thicket_pk" {
		t.Errorf("bc_sable_thicket PK name = %q", sable.PKName)
	}
	if c := sable.Column("shale_thicket"); c == nil || c.Type != "bool" {
		t.Errorf("shale_thicket column = %+v", c)
	}
	if sable.QualifiedName != "public.bc_sable_thicket" {
		t.Errorf("qualified name = %q", sable.QualifiedName)
	}

	content := m.Files[0].Content
	sawArray := false
	for _, tb := range m.Tables {
		if tb.InsertOffset < 0 || tb.InsertOffset > len(content) {
			t.Fatalf("table %s: bad InsertOffset %d", tb.Name, tb.InsertOffset)
		}
		if tb.InsertOffset > 0 && content[tb.InsertOffset-1] != '\n' {
			t.Errorf("table %s: InsertOffset %d is not at start of line", tb.Name, tb.InsertOffset)
		}
		for _, c := range tb.Columns {
			if c.Type == "_text" {
				sawArray = true
			}
		}
	}
	if !sawArray {
		t.Error("no _text columns found, expected 2 in fixture")
	}
}

func TestParseSynthetic(t *testing.T) {
	sql := `CREATE TABLE ord (id text NOT NULL, num int4, amount numeric(10,2), tags text[]);
ALTER TABLE ord ADD CONSTRAINT ord_pk PRIMARY KEY (id);
ALTER TABLE ONLY ord ADD CONSTRAINT ord_other_fk FOREIGN KEY (num) REFERENCES other (id);
COMMENT ON TABLE ord IS 'existing table comment';
COMMENT ON COLUMN ord.num IS 'existing column comment';
CREATE TABLE IF NOT EXISTS "Weird Name" ("MixedCol" text, plain int8 PRIMARY KEY);
`
	m, err := ParseFiles([]model.File{{RelPath: "s.sql", Data: []byte(sql)}})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(m.Tables) != 2 {
		t.Fatalf("tables = %d, want 2", len(m.Tables))
	}
	ord := m.Tables[0]
	if ord.Name != "ord" {
		t.Fatalf("first table = %q", ord.Name)
	}
	if len(ord.PKCols) != 1 || ord.PKCols[0] != "id" || ord.PKName != "ord_pk" {
		t.Errorf("ord PK = %v name=%q (want ALTER-added id/ord_pk)", ord.PKCols, ord.PKName)
	}
	if len(ord.ExistingFKCols) != 1 || ord.ExistingFKCols[0] != "num" {
		t.Errorf("ord existing FK cols = %v", ord.ExistingFKCols)
	}
	names := map[string]bool{}
	for _, n := range ord.ConstraintNames {
		names[n] = true
	}
	if !names["ord_pk"] || !names["ord_other_fk"] {
		t.Errorf("ord constraint names = %v", ord.ConstraintNames)
	}
	if c := ord.Column("amount"); c == nil || c.Type != "numeric(10,2)" {
		t.Errorf("amount = %+v", c)
	}
	if c := ord.Column("tags"); c == nil || c.Type != "text[]" {
		t.Errorf("tags = %+v", c)
	}
	if c := ord.Column("id"); c == nil || !c.NotNull {
		t.Errorf("id = %+v, want NOT NULL", c)
	}
	if !m.ExistingTableComments[model.NormKey("ord")] {
		t.Error("existing table comment not recorded")
	}
	if !m.ExistingColumnComments[model.ColKey("ord", "num")] {
		t.Error("existing column comment not recorded")
	}

	weird := m.Tables[1]
	if weird.Name != "Weird Name" {
		t.Fatalf("second table = %q", weird.Name)
	}
	if weird.QualifiedName != `"Weird Name"` {
		t.Errorf("weird qualified = %q", weird.QualifiedName)
	}
	if c := weird.Column("MixedCol"); c == nil {
		t.Error("MixedCol not found (quoted identifier case must be preserved)")
	}
	if len(weird.PKCols) != 1 || weird.PKCols[0] != "plain" {
		t.Errorf("weird PK = %v", weird.PKCols)
	}
}

func TestParseError(t *testing.T) {
	_, err := ParseFiles([]model.File{{RelPath: "bad.sql", Data: []byte("CREATE TABEL nope (")}})
	var fe *model.FatalError
	if !errors.As(err, &fe) || fe.Code != model.FDdlParse {
		t.Fatalf("err = %v, want FatalError %s", err, model.FDdlParse)
	}
}

func TestStripMarkersRoundTrip(t *testing.T) {
	in := "CREATE TABLE a (x text);\nCREATE TABLE b (y text);\n"
	block := "-- apihub-ddl-import:begin table=a\nCOMMENT ON TABLE a IS 'x';\n-- apihub-ddl-import:end table=a\n"
	mid := "CREATE TABLE a (x text);\n" + block + "CREATE TABLE b (y text);\n"
	if got := StripMarkers(mid); got != in {
		t.Errorf("strip mid-file:\n got %q\nwant %q", got, in)
	}
	if got := StripMarkers(in + block); got != in {
		t.Errorf("strip at EOF:\n got %q\nwant %q", got, in)
	}
	if got := StripMarkers(in); got != in {
		t.Errorf("no markers changed content:\n got %q", got)
	}
}

func TestQuoteIdent(t *testing.T) {
	cases := map[string]string{
		"plain":      "plain",
		"snake_case": "snake_case",
		"Mixed":      `"Mixed"`,
		"user":       `"user"`,
		"order":      `"order"`,
		"a b":        `"a b"`,
		`q"q`:        `"q""q"`,
		"col1":       "col1",
	}
	for in, want := range cases {
		if got := QuoteIdent(in); got != want {
			t.Errorf("QuoteIdent(%q) = %q, want %q", in, got, want)
		}
	}
}
