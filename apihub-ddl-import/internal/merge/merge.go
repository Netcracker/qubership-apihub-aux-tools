package merge

import (
	"fmt"
	"strings"

	"apihub-ddl-import/internal/model"
	"apihub-ddl-import/internal/xlsxin"
)

// Merge matches the workbook against the DDL model. It never mutates the DDL —
// the DDL is the source of truth for structure (PKs in particular); the
// workbook only contributes comments, domains and FK marks.
func Merge(ddlm *model.DDLModel, doc *model.CommentsDoc) *model.MergeResult {
	res := &model.MergeResult{
		TableComments:  map[string]string{},
		ColumnComments: map[string]map[string]string{},
		DomainByTable:  map[string]string{},
	}
	res.Warnings = append(res.Warnings, doc.Warnings...)
	st := &res.Stats
	st.DdlFiles = len(ddlm.Files)
	st.TablesDDL = len(ddlm.Tables)
	st.TablesXlsx = len(doc.Tables)
	st.ColumnsXlsx = len(doc.Columns)
	for _, t := range ddlm.Tables {
		st.ColumnsDDL += len(t.Columns)
	}

	// DDL lookups; duplicate table definitions keep the first one.
	ddlByKey := map[string]*model.DDLTable{}
	colsByKey := map[string]map[string]*model.DDLColumn{}
	for _, t := range ddlm.Tables {
		k := model.NormKey(t.Name)
		if _, dup := ddlByKey[k]; dup {
			warn(res, model.Warning{Code: model.WDupDdlTable, Table: t.Name, File: t.File,
				Msg: fmt.Sprintf("table %s is defined more than once in the DDL sources; the first definition wins", t.Name)})
			continue
		}
		ddlByKey[k] = t
		cm := make(map[string]*model.DDLColumn, len(t.Columns))
		for i := range t.Columns {
			cm[model.NormKey(t.Columns[i].Name)] = &t.Columns[i]
		}
		colsByKey[k] = cm
	}

	tblOrder, tblAggs := dedupTables(res, doc)
	colOrder, colAggs := dedupColumns(res, doc)

	// ---- Tables sheet → table comments, domains, TBL_* diffs.
	matchedTables := map[string]bool{}
	for _, k := range tblOrder {
		a := tblAggs[k]
		setDomain(res, k, a.domain)
		t := ddlByKey[k]
		if t == nil {
			warn(res, model.Warning{Code: model.WTblXlsxOnly, Table: a.row.Table,
				Sheet: xlsxin.SheetTables, Row: a.row.Row,
				Msg: fmt.Sprintf("table %q is listed in the workbook but does not exist in the DDL", a.row.Table)})
			continue
		}
		matchedTables[k] = true
		st.TablesMatched++
		switch {
		case a.descConflict:
			// comment suppressed; the DUP_CONFLICT warning already points at the rows
		case a.desc == "":
			st.EmptyTableDesc++
			warn(res, model.Warning{Code: model.WDescEmptyTable, Table: t.Name,
				Sheet: xlsxin.SheetTables, Row: a.row.Row,
				Msg: fmt.Sprintf("table %s has an empty description", t.Name)})
		default:
			res.TableComments[t.Name] = a.desc
			st.TableCommentsGenerated++
			if ddlm.ExistingTableComments[k] {
				warn(res, model.Warning{Code: model.WCommentExists, Table: t.Name,
					Msg: fmt.Sprintf("source DDL already has COMMENT ON TABLE %s; the generated comment is appended after it and wins", t.Name)})
			}
		}
	}
	for _, t := range ddlm.Tables {
		k := model.NormKey(t.Name)
		if ddlByKey[k] != t {
			continue
		}
		if !matchedTables[k] {
			warn(res, model.Warning{Code: model.WTblDdlOnly, Table: t.Name, File: t.File,
				Msg: fmt.Sprintf("table %s exists in the DDL but has no row in the workbook", t.Name)})
		}
	}

	// ---- Specifications sheet → column comments, PK marks, FK requests.
	type fkRequest struct {
		t   *model.DDLTable
		col *model.DDLColumn
		agg *colAgg
	}
	var fkReqs []fkRequest
	seenCols := map[string]bool{}
	excelPK := map[string]map[string]bool{}
	specTouched := map[string]bool{}

	for _, k := range colOrder {
		a := colAggs[k]
		tk := model.NormKey(a.row.Table)
		setDomain(res, tk, a.domain)
		seenCols[k] = true
		t := ddlByKey[tk]
		var col *model.DDLColumn
		if t != nil {
			specTouched[tk] = true
			col = colsByKey[tk][model.NormKey(a.row.Column)]
		}
		if t == nil || col == nil {
			warn(res, model.Warning{Code: model.WColXlsxOnly, Table: a.row.Table, Column: a.row.Column,
				Sheet: xlsxin.SheetSpecs, Row: a.row.Row,
				Msg: fmt.Sprintf("column %s.%s is listed in the workbook but does not exist in the DDL", a.row.Table, a.row.Column)})
			continue
		}
		st.ColumnsMatched++
		switch {
		case a.descConflict:
		case a.desc == "":
			st.EmptyColumnDesc++
			warn(res, model.Warning{Code: model.WDescEmptyColumn, Table: t.Name, Column: col.Name,
				Sheet: xlsxin.SheetSpecs, Row: a.row.Row,
				Msg: fmt.Sprintf("column %s.%s has an empty description", t.Name, col.Name)})
		default:
			if res.ColumnComments[t.Name] == nil {
				res.ColumnComments[t.Name] = map[string]string{}
			}
			res.ColumnComments[t.Name][col.Name] = a.desc
			st.ColumnCommentsGenerated++
			if ddlm.ExistingColumnComments[model.ColKey(t.Name, col.Name)] {
				warn(res, model.Warning{Code: model.WCommentExists, Table: t.Name, Column: col.Name,
					Msg: fmt.Sprintf("source DDL already has COMMENT ON COLUMN %s.%s; the generated comment is appended after it and wins", t.Name, col.Name)})
			}
		}
		if a.isPK == "Y" {
			if excelPK[tk] == nil {
				excelPK[tk] = map[string]bool{}
			}
			excelPK[tk][model.NormKey(col.Name)] = true
		}
		if a.constraint == "PFK" && a.isPK != "Y" {
			warn(res, model.Warning{Code: model.WPfkWithoutPk, Table: t.Name, Column: col.Name,
				Sheet: xlsxin.SheetSpecs, Row: a.row.Row,
				Msg: fmt.Sprintf("column %s.%s is marked PFK but IsPK is not Y — contradictory input", t.Name, col.Name)})
		}
		if a.constraint == "FK" || a.constraint == "PFK" {
			fkReqs = append(fkReqs, fkRequest{t: t, col: col, agg: a})
		}
	}

	// Columns present in the DDL but absent from the workbook. Columns of
	// workbook-missing tables count too (the case-02 "50 vs 12" trap).
	for _, t := range ddlm.Tables {
		k := model.NormKey(t.Name)
		if ddlByKey[k] != t {
			continue
		}
		for i := range t.Columns {
			if !seenCols[model.ColKey(t.Name, t.Columns[i].Name)] {
				warn(res, model.Warning{Code: model.WColDdlOnly, Table: t.Name, Column: t.Columns[i].Name, File: t.File,
					Msg: fmt.Sprintf("column %s.%s exists in the DDL but has no row in the workbook", t.Name, t.Columns[i].Name)})
			}
		}
	}

	// ---- PK comparison: the DDL primary key always wins and is never rewritten;
	// disagreements are reported once per table. A missing mark is only counted
	// for PK columns the workbook actually lists.
	for _, t := range ddlm.Tables {
		k := model.NormKey(t.Name)
		if ddlByKey[k] != t || !specTouched[k] {
			continue
		}
		ddlPK := map[string]bool{}
		for _, c := range t.PKCols {
			ddlPK[model.NormKey(c)] = true
		}
		var extra, missing []string
		for _, c := range t.Columns {
			ck := model.NormKey(c.Name)
			marked := excelPK[k][ck]
			switch {
			case marked && !ddlPK[ck]:
				extra = append(extra, c.Name)
			case !marked && ddlPK[ck] && seenCols[model.ColKey(t.Name, c.Name)]:
				missing = append(missing, c.Name)
			}
		}
		if len(extra) > 0 || len(missing) > 0 {
			st.PKMismatchTables++
			var parts []string
			if len(missing) > 0 {
				parts = append(parts, fmt.Sprintf("PK columns not marked IsPK=Y: %s", strings.Join(missing, ", ")))
			}
			if len(extra) > 0 {
				parts = append(parts, fmt.Sprintf("marked IsPK=Y but not in the DDL PK: %s", strings.Join(extra, ", ")))
			}
			warn(res, model.Warning{Code: model.WPkMismatch, Table: t.Name,
				Msg: fmt.Sprintf("workbook PK marks contradict the DDL primary key (%s) — the DDL key wins: %s",
					strings.Join(t.PKCols, ", "), strings.Join(parts, "; "))})
		}
	}

	// ---- FK resolution by the naming convention.
	rsv := newResolver(ddlByKey, ddlm.Tables)
	takenNames := map[string]map[string]bool{}
	for _, req := range fkReqs {
		existing := false
		for _, c := range req.t.ExistingFKCols {
			if c == req.col.Name {
				existing = true
				break
			}
		}
		if existing {
			warn(res, model.Warning{Code: model.WFkExists, Table: req.t.Name, Column: req.col.Name,
				Msg: fmt.Sprintf("column %s.%s already has a FOREIGN KEY in the source DDL; generation skipped", req.t.Name, req.col.Name)})
			continue
		}
		target, code, why := rsv.resolve(req.t, req.col.Name)
		if target == nil {
			addFkFailure(res, st, code, req.t.Name, req.col.Name, req.agg.row.Row, why)
			continue
		}
		tcol, note, ok := targetColumn(target, req.col.Name)
		if !ok {
			addFkFailure(res, st, model.WFkUnresolved, req.t.Name, req.col.Name, req.agg.row.Row, note)
			continue
		}
		if takenNames[req.t.Name] == nil {
			takenNames[req.t.Name] = map[string]bool{}
			for _, n := range req.t.ConstraintNames {
				takenNames[req.t.Name][n] = true
			}
		}
		res.FKs = append(res.FKs, model.ResolvedFK{
			File:         req.t.File,
			Table:        req.t.Name,
			Column:       req.col.Name,
			TargetTable:  target.Name,
			TargetColumn: tcol,
			Name:         fkName(req.t.Name, req.col.Name, takenNames[req.t.Name]),
			Note:         note,
		})
		st.FKsGenerated++
	}

	st.WarningsTotal = len(res.Warnings)
	return res
}

func addFkFailure(res *model.MergeResult, st *model.Stats, code, table, column string, row int, why string) {
	if code == model.WFkAmbiguous {
		st.FKsAmbiguous++
	} else {
		st.FKsUnresolved++
	}
	warn(res, model.Warning{Code: code, Table: table, Column: column,
		Sheet: xlsxin.SheetSpecs, Row: row,
		Msg: fmt.Sprintf("FK for %s.%s not generated: %s", table, column, why)})
}

func warn(res *model.MergeResult, w model.Warning) {
	res.Warnings = append(res.Warnings, w)
}

func setDomain(res *model.MergeResult, key, domain string) {
	domain = strings.TrimSpace(domain)
	if key == "" || domain == "" {
		return
	}
	if _, ok := res.DomainByTable[key]; ok {
		return
	}
	res.DomainByTable[key] = domain
	for _, d := range res.Domains {
		if d == domain {
			return
		}
	}
	res.Domains = append(res.Domains, domain)
}

// ---- Workbook row de-duplication.
//
// Rows are grouped by normalized key. Rows identical after trimming and flag
// canonicalization are dropped as exact duplicates; rows that disagree are
// reported and, when the disagreement is between two non-empty descriptions,
// no comment is emitted for that entity at all (the tool must not silently
// pick one of two conflicting descriptions).

type tblAgg struct {
	row          model.TableRow
	domain       string
	desc         string
	external     string
	descConflict bool
}

func dedupTables(res *model.MergeResult, doc *model.CommentsDoc) ([]string, map[string]*tblAgg) {
	var order []string
	aggs := map[string]*tblAgg{}
	for _, r := range doc.Tables {
		k := model.NormKey(r.Table)
		if k == "" {
			continue
		}
		a := aggs[k]
		if a == nil {
			aggs[k] = &tblAgg{row: r, domain: r.Domain, desc: r.Desc, external: r.External}
			order = append(order, k)
			continue
		}
		if r.Domain == a.domain && r.Desc == a.desc && r.External == a.external {
			res.Stats.DupExactRows++
			warn(res, model.Warning{Code: model.WDupExact, Table: r.Table,
				Sheet: xlsxin.SheetTables, Row: r.Row,
				Msg: fmt.Sprintf("row %d duplicates row %d for table %q", r.Row, a.row.Row, r.Table)})
			continue
		}
		res.Stats.DupConflicts++
		if r.Desc != a.desc && r.Desc != "" && a.desc != "" {
			a.descConflict = true
		}
		if a.desc == "" {
			a.desc = r.Desc
		}
		if a.domain == "" {
			a.domain = r.Domain
		}
		warn(res, model.Warning{Code: model.WDupConflict, Table: r.Table,
			Sheet: xlsxin.SheetTables, Row: r.Row,
			Msg: fmt.Sprintf("row %d duplicates row %d for table %q with different content (descriptions: %q vs %q); no comment will be emitted for conflicting descriptions",
				r.Row, a.row.Row, r.Table, a.desc, r.Desc)})
	}
	return order, aggs
}

type colAgg struct {
	row          model.ColumnRow
	domain       string
	dataType     string
	desc         string
	deploy       string
	rdb          string
	extdb        string
	isPK         string
	constraint   string
	descConflict bool
}

func dedupColumns(res *model.MergeResult, doc *model.CommentsDoc) ([]string, map[string]*colAgg) {
	var order []string
	aggs := map[string]*colAgg{}
	for _, r := range doc.Columns {
		if model.NormKey(r.Table) == "" || model.NormKey(r.Column) == "" {
			continue
		}
		k := model.ColKey(r.Table, r.Column)
		isPK, okPK := CanonIsPK(r.IsPK)
		if !okPK {
			warn(res, model.Warning{Code: model.WBadFlag, Table: r.Table, Column: r.Column,
				Sheet: xlsxin.SheetSpecs, Row: r.Row,
				Msg: fmt.Sprintf("unrecognized IsPK value %q (treated as empty)", r.IsPK)})
		}
		cons, okC := CanonConstraint(r.Constraint)
		if !okC {
			warn(res, model.Warning{Code: model.WBadFlag, Table: r.Table, Column: r.Column,
				Sheet: xlsxin.SheetSpecs, Row: r.Row,
				Msg: fmt.Sprintf("unrecognized Constraint value %q (treated as empty)", r.Constraint)})
		}
		a := aggs[k]
		if a == nil {
			aggs[k] = &colAgg{row: r, domain: r.Domain, dataType: r.DataType, desc: r.Desc,
				deploy: r.DeployRelease, rdb: r.RDB, extdb: r.ExternalDB, isPK: isPK, constraint: cons}
			order = append(order, k)
			continue
		}
		exact := r.Domain == a.domain && r.DataType == a.dataType && r.Desc == a.desc &&
			r.DeployRelease == a.deploy && r.RDB == a.rdb && r.ExternalDB == a.extdb &&
			isPK == a.isPK && cons == a.constraint
		if exact {
			res.Stats.DupExactRows++
			warn(res, model.Warning{Code: model.WDupExact, Table: r.Table, Column: r.Column,
				Sheet: xlsxin.SheetSpecs, Row: r.Row,
				Msg: fmt.Sprintf("row %d duplicates row %d for column %s.%s", r.Row, a.row.Row, r.Table, r.Column)})
			continue
		}
		res.Stats.DupConflicts++
		if r.Desc != a.desc && r.Desc != "" && a.desc != "" {
			a.descConflict = true
		}
		if a.desc == "" {
			a.desc = r.Desc
		}
		if a.isPK != "Y" && isPK == "Y" {
			a.isPK = "Y"
		}
		if a.constraint == "" {
			a.constraint = cons
		}
		warn(res, model.Warning{Code: model.WDupConflict, Table: r.Table, Column: r.Column,
			Sheet: xlsxin.SheetSpecs, Row: r.Row,
			Msg: fmt.Sprintf("row %d duplicates row %d for column %s.%s with different content (descriptions: %q vs %q); no comment will be emitted for conflicting descriptions",
				r.Row, a.row.Row, r.Table, r.Column, a.desc, r.Desc)})
	}
	return order, aggs
}
