package merge

import (
	"fmt"
	"hash/fnv"
	"regexp"
	"strings"

	"apihub-ddl-import/internal/model"
)

// The workbook's Constraint column carries only PK/FK/PFK — no target table or
// column anywhere. Foreign keys are therefore resolvable ONLY by the
// <table_stem>_id naming convention: column `sable_thicket_id` references the
// table whose name (minus domain prefix and version suffix) is `sable_thicket`.

var versionSuffix = regexp.MustCompile(`_v\d+$`)

// stemView is a table name lowercased with the trailing version suffix removed.
func stemView(name string) string {
	return versionSuffix.ReplaceAllString(strings.ToLower(name), "")
}

// domainOf is the table's domain prefix — the first "_"-separated token.
func domainOf(name string) string {
	name = strings.ToLower(name)
	if i := strings.Index(name, "_"); i > 0 {
		return name[:i]
	}
	return name
}

// restOf is the stem view without the leading domain prefix.
func restOf(name string) string {
	s := stemView(name)
	d := domainOf(s)
	if strings.HasPrefix(s, d+"_") {
		return s[len(d)+1:]
	}
	return s
}

type resolver struct {
	tables []*model.DDLTable
}

func newResolver(canonical map[string]*model.DDLTable, order []*model.DDLTable) *resolver {
	r := &resolver{}
	for _, t := range order {
		if canonical[model.NormKey(t.Name)] == t {
			r.tables = append(r.tables, t)
		}
	}
	return r
}

// resolve finds the FK target for a column of table `from` by the naming
// convention. It returns the target table, or a warning code
// (FK_UNRESOLVED / FK_AMBIGUOUS) with a human explanation.
func (r *resolver) resolve(from *model.DDLTable, colName string) (*model.DDLTable, string, string) {
	lc := strings.ToLower(colName)
	if !strings.HasSuffix(lc, "_id") || len(lc) <= len("_id") {
		return nil, model.WFkUnresolved,
			fmt.Sprintf("column %q does not follow the <table_stem>_id convention", colName)
	}
	stem := strings.TrimSuffix(lc, "_id")

	var cands []*model.DDLTable
	for _, t := range r.tables {
		if t == from {
			continue
		}
		s := stemView(t.Name)
		if s == stem || strings.HasSuffix(s, "_"+stem) {
			cands = append(cands, t)
		}
	}
	if len(cands) == 0 {
		return nil, model.WFkUnresolved,
			fmt.Sprintf("no table matches stem %q for column %q", stem, colName)
	}
	if len(cands) == 1 {
		return cands[0], "", ""
	}

	// Tie-break (a): candidates in the same domain as the referencing table.
	fromDomain := domainOf(from.Name)
	var sameDomain []*model.DDLTable
	for _, t := range cands {
		if domainOf(t.Name) == fromDomain {
			sameDomain = append(sameDomain, t)
		}
	}
	if len(sameDomain) == 1 {
		return sameDomain[0], "", ""
	}
	pool := cands
	if len(sameDomain) > 1 {
		pool = sameDomain
	}
	// Tie-break (b): the stem sits directly after the domain prefix.
	var direct []*model.DDLTable
	for _, t := range pool {
		if restOf(t.Name) == stem {
			direct = append(direct, t)
		}
	}
	if len(direct) == 1 {
		return direct[0], "", ""
	}
	names := make([]string, 0, len(cands))
	for _, t := range cands {
		names = append(names, t.Name)
	}
	return nil, model.WFkAmbiguous,
		fmt.Sprintf("column %q matches several tables (%s) and tie-breaks do not single one out",
			colName, strings.Join(names, ", "))
}

// targetColumn picks the referenced column of the target table: its PK column,
// or for composite PKs the one matching the FK column by name (else the first,
// with an explanatory note).
func targetColumn(t *model.DDLTable, fkCol string) (string, string, bool) {
	switch len(t.PKCols) {
	case 0:
		return "", fmt.Sprintf("target table %s has no primary key to reference", t.Name), false
	case 1:
		return t.PKCols[0], "", true
	}
	nk := model.NormKey(fkCol)
	for _, c := range t.PKCols {
		if model.NormKey(c) == nk {
			return c, fmt.Sprintf("target %s has a composite PK (%s); referenced the name-matching column %s",
				t.Name, strings.Join(t.PKCols, ", "), c), true
		}
	}
	return t.PKCols[0], fmt.Sprintf("target %s has a composite PK (%s); referenced its first column %s",
		t.Name, strings.Join(t.PKCols, ", "), t.PKCols[0]), true
}

// fkName builds a per-table-unique constraint name capped at PostgreSQL's
// 63-byte identifier limit.
func fkName(table, column string, taken map[string]bool) string {
	base := table + "_" + column + "_fk"
	if len(base) > 63 {
		h := fnv.New32a()
		h.Write([]byte(base))
		base = fmt.Sprintf("%.51s_%08x_fk", table+"_"+column, h.Sum32())
	}
	name := base
	for i := 2; taken[name]; i++ {
		suffix := fmt.Sprintf("_%d", i)
		if len(base)+len(suffix) > 63 {
			name = base[:63-len(suffix)] + suffix
		} else {
			name = base + suffix
		}
	}
	taken[name] = true
	return name
}
