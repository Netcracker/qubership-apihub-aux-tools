package ddl

import (
	"fmt"
	"sort"
	"strings"

	"apihub-ddl-import/internal/model"
)

// RenderEnriched splices the generated statements into the (marker-stripped)
// source files and returns the enriched content per file. Layout per file:
//
//	CREATE TABLE public.t (...);
//	-- apihub-ddl-import:begin table=t
//	COMMENT ON TABLE public.t IS '...';
//	COMMENT ON COLUMN public.t.c IS '...';
//	-- apihub-ddl-import:end table=t
//	...
//	-- apihub-ddl-import:begin fk
//	ALTER TABLE public.t ADD CONSTRAINT t_c_fk FOREIGN KEY (c) REFERENCES public.x (id);
//	-- apihub-ddl-import:end fk
//
// Running the tool over its own output is a no-op: the parser strips the marker
// blocks before parsing, so a re-run regenerates the same content.
func RenderEnriched(ddlm *model.DDLModel, res *model.MergeResult) map[string]string {
	fksByFile := map[string][]model.ResolvedFK{}
	for _, fk := range res.FKs {
		fksByFile[fk.File] = append(fksByFile[fk.File], fk)
	}
	qualified := map[string]string{}
	for _, t := range ddlm.Tables {
		if _, ok := qualified[t.Name]; !ok {
			qualified[t.Name] = t.QualifiedName
		}
	}
	out := make(map[string]string, len(ddlm.Files))
	for _, f := range ddlm.Files {
		var ins []insertion
		for _, t := range f.Tables {
			if block := tableBlock(t, res); block != "" {
				ins = append(ins, insertion{off: t.InsertOffset, text: block})
			}
		}
		content := splice(f.Content, ins)
		if fks := fksByFile[f.RelPath]; len(fks) > 0 {
			if content != "" && !strings.HasSuffix(content, "\n") {
				content += "\n"
			}
			content += fkBlock(fks, qualified)
		}
		out[f.RelPath] = content
	}
	return out
}

type insertion struct {
	off  int
	text string
}

func splice(content string, ins []insertion) string {
	if len(ins) == 0 {
		return content
	}
	sort.SliceStable(ins, func(i, j int) bool { return ins[i].off < ins[j].off })
	var b strings.Builder
	prev := 0
	for _, in := range ins {
		b.WriteString(content[prev:in.off])
		if in.off > 0 && content[in.off-1] != '\n' {
			// EOF (or unterminated line): terminate it before starting the block.
			b.WriteByte('\n')
		}
		b.WriteString(in.text)
		prev = in.off
	}
	b.WriteString(content[prev:])
	return b.String()
}

func tableBlock(t *model.DDLTable, res *model.MergeResult) string {
	tableComment, hasTable := res.TableComments[t.Name]
	colComments := res.ColumnComments[t.Name]
	if !hasTable && len(colComments) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s table=%s\n", markerBeginPrefix, t.Name)
	if hasTable {
		fmt.Fprintf(&b, "COMMENT ON TABLE %s IS '%s';\n", t.QualifiedName, escapeSQLString(tableComment))
	}
	for i := range t.Columns {
		col := &t.Columns[i]
		text, ok := colComments[col.Name]
		if !ok {
			continue
		}
		fmt.Fprintf(&b, "COMMENT ON COLUMN %s.%s IS '%s';\n", t.QualifiedName, QuoteIdent(col.Name), escapeSQLString(text))
	}
	fmt.Fprintf(&b, "%s table=%s\n", markerEndPrefix, t.Name)
	return b.String()
}

func fkBlock(fks []model.ResolvedFK, qualified map[string]string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s fk\n", markerBeginPrefix)
	for _, fk := range fks {
		if fk.Note != "" {
			fmt.Fprintf(&b, "-- note: %s\n", fk.Note)
		}
		fmt.Fprintf(&b, "ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s (%s);\n",
			qualifiedName(qualified, fk.Table), QuoteIdent(fk.Name), QuoteIdent(fk.Column),
			qualifiedName(qualified, fk.TargetTable), QuoteIdent(fk.TargetColumn))
	}
	fmt.Fprintf(&b, "%s fk\n", markerEndPrefix)
	return b.String()
}

func qualifiedName(qualified map[string]string, table string) string {
	if q, ok := qualified[table]; ok {
		return q
	}
	return QuoteIdent(table)
}

func escapeSQLString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
