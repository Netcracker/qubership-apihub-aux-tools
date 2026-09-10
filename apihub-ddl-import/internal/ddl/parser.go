// Package ddl parses PostgreSQL DDL sources with the real PostgreSQL grammar
// (libpg_query compiled to WebAssembly via github.com/wasilibs/go-pgquery) and
// renders the generated enrichment statements back into the source files.
package ddl

import (
	"strconv"
	"strings"

	pgast "github.com/pganalyze/pg_query_go/v6"
	pgquery "github.com/wasilibs/go-pgquery"

	"apihub-ddl-import/internal/model"
)

// Marker lines delimit generated blocks; the parser strips them on input so the
// tool can be re-run over its own output.
const (
	markerBeginPrefix = "-- apihub-ddl-import:begin"
	markerEndPrefix   = "-- apihub-ddl-import:end"
)

// StripMarkers removes previously generated enrichment blocks (whole lines from
// a begin marker through the matching end marker inclusive).
func StripMarkers(content string) string {
	if !strings.Contains(content, markerBeginPrefix) {
		return content
	}
	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))
	skip := false
	for _, ln := range lines {
		t := strings.TrimSpace(strings.TrimSuffix(ln, "\r"))
		if !skip && strings.HasPrefix(t, markerBeginPrefix) {
			skip = true
			continue
		}
		if skip {
			if strings.HasPrefix(t, markerEndPrefix) {
				skip = false
			}
			continue
		}
		out = append(out, ln)
	}
	return strings.Join(out, "\n")
}

// ParseFiles parses every source file and assembles the DDL model. Any file the
// PostgreSQL grammar rejects aborts with a F_DDL_PARSE fatal error.
func ParseFiles(files []model.File) (*model.DDLModel, error) {
	m := &model.DDLModel{
		ExistingTableComments:  map[string]bool{},
		ExistingColumnComments: map[string]bool{},
	}
	var alters []*alterInfo
	for _, f := range files {
		pf, fileAlters, err := parseFile(f, m)
		if err != nil {
			return nil, err
		}
		m.Files = append(m.Files, pf)
		m.Tables = append(m.Tables, pf.Tables...)
		alters = append(alters, fileAlters...)
	}
	applyAlters(m, alters)
	return m, nil
}

func parseFile(f model.File, m *model.DDLModel) (*model.DDLFile, []*alterInfo, error) {
	content := StripMarkers(string(f.Data))
	res, err := pgquery.Parse(content)
	if err != nil {
		return nil, nil, model.Fatalf(model.FDdlParse, "%s: %v", f.RelPath, err)
	}
	pf := &model.DDLFile{RelPath: f.RelPath, Content: content}
	var alters []*alterInfo
	for _, raw := range res.Stmts {
		node := raw.GetStmt()
		if node == nil {
			continue
		}
		start := int(raw.GetStmtLocation())
		end := start + int(raw.GetStmtLen())
		if raw.GetStmtLen() == 0 || end > len(content) {
			end = len(content)
		}
		switch {
		case node.GetCreateStmt() != nil:
			t := buildTable(node.GetCreateStmt(), f.RelPath)
			t.StmtStart = start
			t.StmtEnd, t.InsertOffset = insertionPoint(content, end)
			pf.Tables = append(pf.Tables, t)
		case node.GetCommentStmt() != nil:
			recordComment(node.GetCommentStmt(), m)
		case node.GetAlterTableStmt() != nil:
			if ai := readAlter(node.GetAlterTableStmt()); ai != nil {
				alters = append(alters, ai)
			}
		}
	}
	return pf, alters, nil
}

// insertionPoint returns the end of the statement (just past its semicolon when
// present) and the byte offset where an enrichment block may be spliced: the
// start of the line following the semicolon.
func insertionPoint(content string, stmtTextEnd int) (stmtEnd, insertAt int) {
	i := stmtTextEnd
	for i < len(content) {
		switch content[i] {
		case ' ', '\t', '\r', '\n':
			i++
			continue
		}
		break
	}
	stmtEnd = stmtTextEnd
	if i < len(content) && content[i] == ';' {
		stmtEnd = i + 1
	}
	nl := strings.IndexByte(content[stmtEnd:], '\n')
	if nl < 0 {
		return stmtEnd, len(content)
	}
	return stmtEnd, stmtEnd + nl + 1
}

func buildTable(cs *pgast.CreateStmt, file string) *model.DDLTable {
	rel := cs.GetRelation()
	t := &model.DDLTable{
		Schema: rel.GetSchemaname(),
		Name:   rel.GetRelname(),
		File:   file,
	}
	if t.Schema != "" {
		t.QualifiedName = QuoteIdent(t.Schema) + "." + QuoteIdent(t.Name)
	} else {
		t.QualifiedName = QuoteIdent(t.Name)
	}
	elts := make([]*pgast.Node, 0, len(cs.GetTableElts())+len(cs.GetConstraints()))
	elts = append(elts, cs.GetTableElts()...)
	elts = append(elts, cs.GetConstraints()...)
	pos := 0
	for _, elt := range elts {
		if cd := elt.GetColumnDef(); cd != nil {
			col := model.DDLColumn{
				Name:    cd.GetColname(),
				Type:    renderTypeName(cd.GetTypeName()),
				NotNull: cd.GetIsNotNull(),
				Pos:     pos,
			}
			pos++
			for _, cn := range cd.GetConstraints() {
				con := cn.GetConstraint()
				if con == nil {
					continue
				}
				if con.GetConname() != "" {
					t.ConstraintNames = append(t.ConstraintNames, con.GetConname())
				}
				switch con.GetContype() {
				case pgast.ConstrType_CONSTR_PRIMARY:
					t.PKCols = append(t.PKCols, col.Name)
					if con.GetConname() != "" {
						t.PKName = con.GetConname()
					}
				case pgast.ConstrType_CONSTR_NOTNULL:
					col.NotNull = true
				case pgast.ConstrType_CONSTR_FOREIGN:
					t.ExistingFKCols = append(t.ExistingFKCols, col.Name)
				}
			}
			t.Columns = append(t.Columns, col)
			continue
		}
		con := elt.GetConstraint()
		if con == nil {
			continue
		}
		if con.GetConname() != "" {
			t.ConstraintNames = append(t.ConstraintNames, con.GetConname())
		}
		switch con.GetContype() {
		case pgast.ConstrType_CONSTR_PRIMARY:
			t.PKName = con.GetConname()
			t.PKCols = stringListVals(con.GetKeys())
		case pgast.ConstrType_CONSTR_FOREIGN:
			t.ExistingFKCols = append(t.ExistingFKCols, stringListVals(con.GetFkAttrs())...)
		}
	}
	return t
}

// alterInfo carries constraints added via ALTER TABLE so they can be attached to
// the owning table after all files are parsed (pg_dump declares PKs/FKs that way).
type alterInfo struct {
	table  string
	names  []string
	fkCols []string
	pkName string
	pkCols []string
}

func readAlter(at *pgast.AlterTableStmt) *alterInfo {
	rel := at.GetRelation()
	if rel == nil {
		return nil
	}
	ai := &alterInfo{table: rel.GetRelname()}
	for _, c := range at.GetCmds() {
		cmd := c.GetAlterTableCmd()
		if cmd == nil || cmd.GetSubtype() != pgast.AlterTableType_AT_AddConstraint {
			continue
		}
		con := cmd.GetDef().GetConstraint()
		if con == nil {
			continue
		}
		if con.GetConname() != "" {
			ai.names = append(ai.names, con.GetConname())
		}
		switch con.GetContype() {
		case pgast.ConstrType_CONSTR_PRIMARY:
			ai.pkName = con.GetConname()
			ai.pkCols = stringListVals(con.GetKeys())
		case pgast.ConstrType_CONSTR_FOREIGN:
			ai.fkCols = append(ai.fkCols, stringListVals(con.GetFkAttrs())...)
		}
	}
	if len(ai.names) == 0 && len(ai.fkCols) == 0 && len(ai.pkCols) == 0 {
		return nil
	}
	return ai
}

func applyAlters(m *model.DDLModel, alters []*alterInfo) {
	if len(alters) == 0 {
		return
	}
	byName := map[string]*model.DDLTable{}
	for _, t := range m.Tables {
		byName[t.Name] = t
	}
	for _, ai := range alters {
		t := byName[ai.table]
		if t == nil {
			continue
		}
		t.ConstraintNames = append(t.ConstraintNames, ai.names...)
		t.ExistingFKCols = append(t.ExistingFKCols, ai.fkCols...)
		if len(ai.pkCols) > 0 && len(t.PKCols) == 0 {
			t.PKName = ai.pkName
			t.PKCols = ai.pkCols
		}
	}
}

func recordComment(com *pgast.CommentStmt, m *model.DDLModel) {
	names := objectNames(com.GetObject())
	switch com.GetObjtype() {
	case pgast.ObjectType_OBJECT_TABLE:
		if len(names) > 0 {
			m.ExistingTableComments[model.NormKey(names[len(names)-1])] = true
		}
	case pgast.ObjectType_OBJECT_COLUMN:
		if len(names) >= 2 {
			table := names[len(names)-2]
			column := names[len(names)-1]
			m.ExistingColumnComments[model.ColKey(table, column)] = true
		}
	}
}

func objectNames(n *pgast.Node) []string {
	if n == nil {
		return nil
	}
	if l := n.GetList(); l != nil {
		out := make([]string, 0, len(l.GetItems()))
		for _, it := range l.GetItems() {
			if s := it.GetString_(); s != nil {
				out = append(out, s.GetSval())
			}
		}
		return out
	}
	if s := n.GetString_(); s != nil {
		return []string{s.GetSval()}
	}
	return nil
}

func stringListVals(nodes []*pgast.Node) []string {
	out := make([]string, 0, len(nodes))
	for _, n := range nodes {
		if s := n.GetString_(); s != nil {
			out = append(out, s.GetSval())
		}
	}
	return out
}

func renderTypeName(tn *pgast.TypeName) string {
	if tn == nil {
		return ""
	}
	parts := make([]string, 0, 2)
	for _, n := range tn.GetNames() {
		if s := n.GetString_(); s != nil {
			if s.GetSval() == "pg_catalog" {
				continue
			}
			parts = append(parts, s.GetSval())
		}
	}
	name := strings.Join(parts, ".")
	if mods := tn.GetTypmods(); len(mods) > 0 {
		vals := make([]string, 0, len(mods))
		for _, mn := range mods {
			if ac := mn.GetAConst(); ac != nil {
				if iv := ac.GetIval(); iv != nil {
					vals = append(vals, strconv.Itoa(int(iv.GetIval())))
				}
			}
		}
		if len(vals) > 0 {
			name += "(" + strings.Join(vals, ",") + ")"
		}
	}
	if len(tn.GetArrayBounds()) > 0 {
		name += "[]"
	}
	return name
}

// reservedIdents are common reserved words that must be quoted when used as
// identifiers in generated statements.
var reservedIdents = map[string]bool{
	"all": true, "and": true, "any": true, "asc": true, "case": true, "cast": true,
	"check": true, "column": true, "constraint": true, "create": true, "current_date": true,
	"current_time": true, "current_timestamp": true, "current_user": true, "default": true,
	"desc": true, "distinct": true, "do": true, "else": true, "end": true, "false": true,
	"for": true, "foreign": true, "from": true, "grant": true, "group": true, "having": true,
	"in": true, "initially": true, "intersect": true, "into": true, "leading": true,
	"limit": true, "localtime": true, "localtimestamp": true, "not": true, "null": true,
	"offset": true, "on": true, "only": true, "or": true, "order": true, "placing": true,
	"primary": true, "references": true, "returning": true, "select": true, "session_user": true,
	"some": true, "table": true, "then": true, "to": true, "trailing": true, "true": true,
	"union": true, "unique": true, "user": true, "using": true, "when": true, "where": true,
	"window": true, "with": true,
}

// QuoteIdent renders an identifier for generated SQL, double-quoting it when it
// is not a plain lower-case identifier or collides with a reserved word.
func QuoteIdent(s string) string {
	plain := s != "" && !reservedIdents[s]
	if plain {
		for i, r := range s {
			lower := r >= 'a' && r <= 'z'
			digit := r >= '0' && r <= '9'
			if !(lower || r == '_' || (i > 0 && (digit || r == '$'))) {
				plain = false
				break
			}
		}
	}
	if plain {
		return s
	}
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}
