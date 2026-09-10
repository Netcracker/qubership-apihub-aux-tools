// Package enrich post-processes the two DDL xlsx exports downloaded from
// APIHUB: it guarantees a filled "Group" column (domain from the workbook) and
// adds the computed "Analytics Severity" column to the changes report.
//
// The logic is header-NAME-driven, not cell-letter-driven, so it survives all
// three layout generations: pre-Group backends (no Group column → appended),
// the current per-entity count layout (Group at H/L), and a possible future
// per-change layout with a "Change Severity" column.
package enrich

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"

	"apihub-ddl-import/internal/model"
)

// DdlSheet is the data sheet name of both DDL reports (backend view/Excel.go).
const DdlSheet = "DDL"

// DefaultDataTypeRegex detects data-type changes in change descriptions; any
// match is treated as breaking regardless of the reported severity (the
// documented Analytics Severity exception). Overridable via config.
var DefaultDataTypeRegex = regexp.MustCompile(`(?i)\b(data\s+)?type\b.{0,60}\b(chang|updat)|\b(chang|updat)\w*\b.{0,60}\b(data\s+)?type\b`)

// Change is one per-entity change (mirrors the APIHUB response).
type Change struct {
	Description string
	Severity    string
}

// Options parameterize the enrichment.
type Options struct {
	// DomainByTable maps model.NormKey(table name) → workbook domain.
	DomainByTable map[string]string
	// EntityIDByName maps model.NormKey(entity name) → ddlEntityId (changes report).
	EntityIDByName map[string]string
	// FetchChanges loads the per-change list of one entity; nil disables the
	// per-change path and forces the count-columns fallback.
	FetchChanges func(ddlEntityID string) ([]Change, error)
	// DataTypeRegex overrides DefaultDataTypeRegex when non-nil.
	DataTypeRegex *regexp.Regexp
}

// Info reports what the enrichment did.
type Info struct {
	Rows              int
	GroupFilled       int
	GroupAppended     bool
	AnalyticsFallback bool
	Warnings          []model.Warning
}

func (o Options) regex() *regexp.Regexp {
	if o.DataTypeRegex != nil {
		return o.DataTypeRegex
	}
	return DefaultDataTypeRegex
}

type sheetCtx struct {
	f       *excelize.File
	sheet   string
	rows    [][]string
	headers map[string]int // lower(trimmed header) → 0-based column index
	width   int            // number of header cells
}

func openSheet(data []byte) (*sheetCtx, *Info, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, nil, fmt.Errorf("open export workbook: %w", err)
	}
	sheet := ""
	for _, s := range f.GetSheetList() {
		if strings.EqualFold(strings.TrimSpace(s), DdlSheet) {
			sheet = s
			break
		}
	}
	info := &Info{}
	if sheet == "" {
		info.Warnings = append(info.Warnings, model.Warning{Code: model.WExportLayoutUnknown,
			Msg: fmt.Sprintf("export has no %q sheet (sheets: %s) — saved unmodified", DdlSheet, strings.Join(f.GetSheetList(), ", "))})
		f.Close()
		return nil, info, nil
	}
	rows, err := f.GetRows(sheet)
	if err != nil || len(rows) == 0 {
		info.Warnings = append(info.Warnings, model.Warning{Code: model.WExportLayoutUnknown,
			Msg: "export DDL sheet has no header row — saved unmodified"})
		f.Close()
		return nil, info, nil
	}
	headers := map[string]int{}
	for i, h := range rows[0] {
		headers[strings.ToLower(strings.TrimSpace(h))] = i
	}
	return &sheetCtx{f: f, sheet: sheet, rows: rows, headers: headers, width: len(rows[0])}, info, nil
}

func (s *sheetCtx) cell(rowIdx, colIdx int) string {
	if rowIdx >= len(s.rows) {
		return ""
	}
	row := s.rows[rowIdx]
	if colIdx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[colIdx])
}

func (s *sheetCtx) set(rowIdx, colIdx int, value any) error {
	name, err := excelize.CoordinatesToCellName(colIdx+1, rowIdx+1)
	if err != nil {
		return err
	}
	return s.f.SetCellValue(s.sheet, name, value)
}

// addColumn appends a header cell in the next free column, copying the style of
// the first header cell, and returns its 0-based index.
func (s *sheetCtx) addColumn(title string) (int, error) {
	idx := s.width
	s.width++
	if err := s.set(0, idx, title); err != nil {
		return 0, err
	}
	if style, err := s.f.GetCellStyle(s.sheet, "A1"); err == nil {
		name, _ := excelize.CoordinatesToCellName(idx+1, 1)
		_ = s.f.SetCellStyle(s.sheet, name, name, style)
	}
	if col, err := excelize.ColumnNumberToName(idx + 1); err == nil {
		_ = s.f.SetColWidth(s.sheet, col, col, 30)
	}
	return idx, nil
}

func (s *sheetCtx) save() ([]byte, error) {
	defer s.f.Close()
	buf, err := s.f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// fillGroup ensures the Group column exists and is filled from the domain map.
// Backend-filled non-empty cells are kept; a cell whose groups do not include
// the workbook domain is reported as a mismatch.
func fillGroup(s *sheetCtx, nameIdx int, opts Options, info *Info) (int, error) {
	groupIdx, had := s.headers["group"]
	if !had {
		var err error
		groupIdx, err = s.addColumn("Group")
		if err != nil {
			return 0, err
		}
		info.GroupAppended = true
	}
	for i := 1; i < len(s.rows); i++ {
		name := s.cell(i, nameIdx)
		if name == "" {
			continue
		}
		domain := opts.DomainByTable[model.NormKey(name)]
		cur := ""
		if had {
			cur = s.cell(i, groupIdx)
		}
		switch {
		case cur == "" && domain != "":
			if err := s.set(i, groupIdx, domain); err != nil {
				return 0, err
			}
			info.GroupFilled++
		case cur != "" && domain != "" && !containsGroup(cur, domain):
			info.Warnings = append(info.Warnings, model.Warning{
				Code: model.WExportGroupMismatch, Table: name,
				Msg: fmt.Sprintf("export row for %q has Group %q which does not include the workbook domain %q — backend value kept", name, cur, domain),
			})
		}
	}
	return groupIdx, nil
}

func containsGroup(cell, domain string) bool {
	for _, part := range strings.Split(cell, ",") {
		if strings.EqualFold(strings.TrimSpace(part), domain) {
			return true
		}
	}
	return false
}

// Entities enriches the entities export: Group column only.
func Entities(data []byte, opts Options) ([]byte, *Info, error) {
	s, info, err := openSheet(data)
	if err != nil {
		return nil, nil, err
	}
	if s == nil {
		return data, info, nil
	}
	nameIdx, ok := s.headers["name"]
	if !ok {
		info.Warnings = append(info.Warnings, model.Warning{Code: model.WExportLayoutUnknown,
			Msg: "entities export has no Name column — saved unmodified"})
		s.f.Close()
		return data, info, nil
	}
	info.Rows = len(s.rows) - 1
	if _, err := fillGroup(s, nameIdx, opts, info); err != nil {
		s.f.Close()
		return nil, info, err
	}
	out, err := s.save()
	if err != nil {
		return nil, info, err
	}
	return out, info, nil
}

// severity ranks for the "worst wins" rule: breaking > non-breaking > unclassified.
func rank(analytics string) int {
	switch analytics {
	case "breaking":
		return 3
	case "non-breaking":
		return 2
	default:
		return 1
	}
}

// mapAnalytics applies the documented mapping (requiring attention is the
// display alias of semi-breaking; deprecated is assumed breaking) plus the
// data-type exception.
func mapAnalytics(severity, description string, re *regexp.Regexp) string {
	if re != nil && description != "" && re.MatchString(description) {
		return "breaking"
	}
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "breaking", "semi-breaking", "annotation", "deprecated", "requiring attention":
		return "breaking"
	case "non-breaking":
		return "non-breaking"
	default:
		return "unclassified"
	}
}

var countHeaders = []string{"breaking", "semi-breaking", "deprecated", "non-breaking", "annotation", "unclassified"}

// Changes enriches the changes export: Group column plus Analytics Severity.
func Changes(data []byte, opts Options) ([]byte, *Info, error) {
	s, info, err := openSheet(data)
	if err != nil {
		return nil, nil, err
	}
	if s == nil {
		return data, info, nil
	}
	nameIdx, okName := s.headers["name"]
	sevIdx, perChange := s.headers["change severity"]
	countIdx := map[string]int{}
	haveCounts := true
	for _, h := range countHeaders {
		idx, ok := s.headers[h]
		if !ok {
			haveCounts = false
			break
		}
		countIdx[h] = idx
	}
	if !okName || (!perChange && !haveCounts) {
		info.Warnings = append(info.Warnings, model.Warning{Code: model.WExportLayoutUnknown,
			Msg: "changes export layout not recognized (no Name + no severity columns) — saved unmodified"})
		s.f.Close()
		return data, info, nil
	}
	info.Rows = len(s.rows) - 1

	if _, err := fillGroup(s, nameIdx, opts, info); err != nil {
		s.f.Close()
		return nil, info, err
	}
	analyticsIdx, err := s.addColumn("Analytics Severity")
	if err != nil {
		s.f.Close()
		return nil, info, err
	}

	re := opts.regex()
	descIdx, hasDesc := s.headers["change description"]
	for i := 1; i < len(s.rows); i++ {
		name := s.cell(i, nameIdx)
		if name == "" {
			continue
		}
		var value string
		if perChange {
			desc := ""
			if hasDesc {
				desc = s.cell(i, descIdx)
			}
			value = mapAnalytics(s.cell(i, sevIdx), desc, re)
		} else {
			value = s.analyticsFromEntity(i, name, countIdx, opts, re, info)
		}
		if err := s.set(i, analyticsIdx, value); err != nil {
			s.f.Close()
			return nil, info, err
		}
	}
	out, err := s.save()
	if err != nil {
		return nil, info, err
	}
	return out, info, nil
}

// analyticsFromEntity computes the row value for the per-entity count layout:
// preferably from the per-change API (worst mapped severity, data-type
// exception applied), falling back to the count columns.
func (s *sheetCtx) analyticsFromEntity(rowIdx int, name string, countIdx map[string]int, opts Options, re *regexp.Regexp, info *Info) string {
	if opts.FetchChanges != nil {
		if id := opts.EntityIDByName[model.NormKey(name)]; id != "" {
			if changes, err := opts.FetchChanges(id); err == nil && len(changes) > 0 {
				worst := "unclassified"
				for _, ch := range changes {
					v := mapAnalytics(ch.Severity, ch.Description, re)
					if rank(v) > rank(worst) {
						worst = v
					}
				}
				return worst
			}
		}
	}
	info.AnalyticsFallback = true
	n := func(h string) int {
		v, _ := strconv.Atoi(s.cell(rowIdx, countIdx[h]))
		return v
	}
	switch {
	case n("breaking")+n("semi-breaking")+n("annotation")+n("deprecated") > 0:
		return "breaking"
	case n("non-breaking") > 0:
		return "non-breaking"
	default:
		return "unclassified"
	}
}
