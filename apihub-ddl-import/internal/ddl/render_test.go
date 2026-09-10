package ddl

import (
	"strings"
	"testing"

	"apihub-ddl-import/internal/model"
)

func TestRenderEnrichedAndIdempotency(t *testing.T) {
	src := "CREATE TABLE public.acc (id text NOT NULL, owner_id text, CONSTRAINT acc_pk PRIMARY KEY (id));\n" +
		"CREATE TABLE public.owner (id text, CONSTRAINT owner_pk PRIMARY KEY (id));\n"
	files := []model.File{{RelPath: "s.sql", Data: []byte(src)}}
	m, err := ParseFiles(files)
	if err != nil {
		t.Fatal(err)
	}
	res := &model.MergeResult{
		TableComments: map[string]string{"acc": "Account's table"},
		ColumnComments: map[string]map[string]string{
			"acc": {"id": "Identifier with 'quotes'", "owner_id": "Owner ref"},
		},
		FKs: []model.ResolvedFK{{
			File: "s.sql", Table: "acc", Column: "owner_id",
			TargetTable: "owner", TargetColumn: "id", Name: "acc_owner_id_fk",
		}},
	}
	out := RenderEnriched(m, res)
	got := out["s.sql"]

	for _, want := range []string{
		"COMMENT ON TABLE public.acc IS 'Account''s table';",
		"COMMENT ON COLUMN public.acc.id IS 'Identifier with ''quotes''';",
		"COMMENT ON COLUMN public.acc.owner_id IS 'Owner ref';",
		"ALTER TABLE public.acc ADD CONSTRAINT acc_owner_id_fk FOREIGN KEY (owner_id) REFERENCES public.owner (id);",
		"-- apihub-ddl-import:begin table=acc",
		"-- apihub-ddl-import:end table=acc",
		"-- apihub-ddl-import:begin fk",
		"-- apihub-ddl-import:end fk",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("enriched output missing %q\n---\n%s", want, got)
		}
	}
	// The table block must sit right after the acc statement, before owner's CREATE.
	if strings.Index(got, "begin table=acc") > strings.Index(got, "CREATE TABLE public.owner") {
		t.Error("table block is not spliced after its statement")
	}
	// Column comment order must follow the DDL column order.
	if strings.Index(got, "acc.id IS") > strings.Index(got, "acc.owner_id IS") {
		t.Error("column comments are not in DDL order")
	}

	// Marker strip restores the original bytes exactly.
	if stripped := StripMarkers(got); stripped != src {
		t.Errorf("StripMarkers(enriched) != original\n got: %q\nwant: %q", stripped, src)
	}

	// Re-running the tool over its own output must be a no-op.
	m2, err := ParseFiles([]model.File{{RelPath: "s.sql", Data: []byte(got)}})
	if err != nil {
		t.Fatal(err)
	}
	out2 := RenderEnriched(m2, res)
	if out2["s.sql"] != got {
		t.Errorf("second render differs from first:\n---1---\n%s\n---2---\n%s", got, out2["s.sql"])
	}
}

func TestRenderNoTrailingNewlineAtEOF(t *testing.T) {
	src := "CREATE TABLE t1 (id text PRIMARY KEY);" // no trailing newline
	m, err := ParseFiles([]model.File{{RelPath: "e.sql", Data: []byte(src)}})
	if err != nil {
		t.Fatal(err)
	}
	res := &model.MergeResult{
		TableComments:  map[string]string{"t1": "x"},
		ColumnComments: map[string]map[string]string{},
	}
	out := RenderEnriched(m, res)
	got := out["e.sql"]
	if !strings.Contains(got, ";\n-- apihub-ddl-import:begin table=t1\n") {
		t.Errorf("statement line must be terminated before the block:\n%s", got)
	}
	// Fixed point: render(parse(render)) == render.
	m2, _ := ParseFiles([]model.File{{RelPath: "e.sql", Data: []byte(got)}})
	if out2 := RenderEnriched(m2, res); out2["e.sql"] != got {
		t.Errorf("not a fixed point:\n---1---\n%s\n---2---\n%s", got, out2["e.sql"])
	}
}

func TestRenderSkipsTablesWithoutComments(t *testing.T) {
	src := "CREATE TABLE a (x text);\nCREATE TABLE b (y text);\n"
	m, err := ParseFiles([]model.File{{RelPath: "s.sql", Data: []byte(src)}})
	if err != nil {
		t.Fatal(err)
	}
	res := &model.MergeResult{
		TableComments:  map[string]string{"a": "only a"},
		ColumnComments: map[string]map[string]string{},
	}
	got := RenderEnriched(m, res)["s.sql"]
	if strings.Contains(got, "table=b") {
		t.Errorf("table b has no comments and must get no block:\n%s", got)
	}
}
