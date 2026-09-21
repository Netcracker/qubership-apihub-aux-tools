// Package report renders the merge/pipeline outcome to the console (logx) and
// to report.md / report.json in the output directory.
package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"apihub-ddl-import/internal/logx"
	"apihub-ddl-import/internal/model"
)

const ToolName = "apihub-ddl-import"

// ToolVersion is stamped by the release build via -ldflags.
var ToolVersion = "dev"

// Inputs echoes what the run consumed.
type Inputs struct {
	DDLSource       string `json:"ddlSource"`
	CommentsSource  string `json:"commentsSource"`
	PackageID       string `json:"packageId,omitempty"`
	Version         string `json:"version,omitempty"`
	PreviousVersion string `json:"previousVersion,omitempty"`
	Status          string `json:"status,omitempty"`
	DryRun          bool   `json:"dryRun"`
}

type FatalInfo struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
}

type PublishInfo struct {
	PublishID   string  `json:"publishId,omitempty"`
	Status      string  `json:"status,omitempty"`
	Message     string  `json:"message,omitempty"`
	DurationSec float64 `json:"durationSec,omitempty"`
}

type GroupsInfo struct {
	APIAvailable bool     `json:"apiAvailable"`
	Created      []string `json:"created,omitempty"`
	Updated      []string `json:"updated,omitempty"`
	Failed       []string `json:"failed,omitempty"`
	Skipped      bool     `json:"skipped,omitempty"`
}

type ExportFile struct {
	Path              string `json:"path"`
	Rows              int    `json:"rows"`
	GroupFilled       int    `json:"groupFilled"`
	GroupAppended     bool   `json:"groupAppended,omitempty"`
	AnalyticsFallback bool   `json:"analyticsFallback,omitempty"`
	// Enriched is false when --skip-enrichment kept APIHUB's raw export
	// (no Group / Analytics Severity columns added).
	Enriched bool `json:"enriched"`
}

type ExportsInfo struct {
	Entities *ExportFile `json:"entities,omitempty"`
	Changes  *ExportFile `json:"changes,omitempty"`
}

// Report is the machine-readable run summary (report.json).
type Report struct {
	Tool        string          `json:"tool"`
	ToolVersion string          `json:"toolVersion"`
	GeneratedAt time.Time       `json:"generatedAt"`
	Inputs      Inputs          `json:"inputs"`
	Stats       model.Stats     `json:"stats"`
	Warnings    []model.Warning `json:"warnings"`
	Fatal       *FatalInfo      `json:"fatal,omitempty"`
	Publish     *PublishInfo    `json:"publish,omitempty"`
	Groups      *GroupsInfo     `json:"groups,omitempty"`
	Exports     *ExportsInfo    `json:"exports,omitempty"`
}

func New(inputs Inputs) *Report {
	return &Report{Tool: ToolName, ToolVersion: ToolVersion, GeneratedAt: time.Now().UTC(), Inputs: inputs}
}

func (r *Report) SetMerge(res *model.MergeResult) {
	r.Stats = res.Stats
	r.Warnings = res.Warnings
}

func (r *Report) SetFatal(fe *model.FatalError) {
	r.Fatal = &FatalInfo{Code: fe.Code, Msg: fe.Msg}
}

// AddWarning appends a pipeline-stage warning (groups/exports) after the merge.
func (r *Report) AddWarning(w model.Warning) {
	r.Warnings = append(r.Warnings, w)
	r.Stats.WarningsTotal = len(r.Warnings)
}

// categoryOrder fixes the display order of warning categories.
var categoryOrder = []string{
	model.WTblDdlOnly, model.WTblXlsxOnly, model.WColDdlOnly, model.WColXlsxOnly,
	model.WDescEmptyTable, model.WDescEmptyColumn,
	model.WPkMismatch,
	model.WFkUnresolved, model.WFkAmbiguous, model.WFkExists, model.WPfkWithoutPk,
	model.WDupExact, model.WDupConflict, model.WBadFlag,
	model.WSheetEmpty, model.WCommentExists, model.WDupDdlTable,
	model.WGroupsApiUnavailable, model.WGroupCreateFailed,
	model.WExportUnavailable, model.WExportLayoutUnknown, model.WExportGroupMismatch,
	model.WAnalyticsFallback, model.WDescriptionsMissing,
}

func (r *Report) byCategory() ([]string, map[string][]model.Warning) {
	byCode := map[string][]model.Warning{}
	for _, w := range r.Warnings {
		byCode[w.Code] = append(byCode[w.Code], w)
	}
	var order []string
	seen := map[string]bool{}
	for _, c := range categoryOrder {
		if len(byCode[c]) > 0 {
			order = append(order, c)
			seen[c] = true
		}
	}
	for _, w := range r.Warnings {
		if !seen[w.Code] {
			order = append(order, w.Code)
			seen[w.Code] = true
		}
	}
	return order, byCode
}

// consoleDetailsPerCategory caps per-category console detail lines; the full
// lists are always in report.md / report.json.
const consoleDetailsPerCategory = 20

// PrintSummary writes the human summary to the console.
func (r *Report) PrintSummary() {
	logx.Section("Merge report")
	st := r.Stats
	logx.Notef("DDL: %d file(s), %d tables, %d columns; workbook: %d table rows, %d column rows",
		st.DdlFiles, st.TablesDDL, st.ColumnsDDL, st.TablesXlsx, st.ColumnsXlsx)
	logx.Notef("Matched: %d tables, %d columns; comments generated: %d table, %d column; FKs: %d generated, %d unresolved, %d ambiguous",
		st.TablesMatched, st.ColumnsMatched, st.TableCommentsGenerated, st.ColumnCommentsGenerated,
		st.FKsGenerated, st.FKsUnresolved, st.FKsAmbiguous)

	order, byCode := r.byCategory()
	if len(order) > 0 {
		logx.Infof("Warnings by category:")
		for _, code := range order {
			logx.Warnf("  %-24s %d", code, len(byCode[code]))
		}
		for _, code := range order {
			ws := byCode[code]
			logx.Infof("%s:", code)
			for i, w := range ws {
				if i >= consoleDetailsPerCategory {
					logx.Infof("  … and %d more (see report.md)", len(ws)-consoleDetailsPerCategory)
					break
				}
				logx.Infof("  %s", w.Msg)
			}
		}
	}

	switch {
	case r.Fatal != nil:
		logx.Errorf("FATAL %s: %s", r.Fatal.Code, r.Fatal.Msg)
	case len(r.Warnings) == 0:
		logx.Okf("Merge clean: no warnings")
	default:
		logx.Warnf("Merge finished with %d warning(s)", len(r.Warnings))
	}
}

// WriteFiles writes report.json and report.md into dir.
func (r *Report) WriteFiles(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	js, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "report.json"), append(js, '\n'), 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "report.md"), []byte(r.Markdown()), 0o644)
}

// Markdown renders the full report.
func (r *Report) Markdown() string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s report\n\n", ToolName)
	fmt.Fprintf(&b, "- Generated: %s (%s %s)\n", r.GeneratedAt.Format(time.RFC3339), ToolName, r.ToolVersion)
	fmt.Fprintf(&b, "- DDL source: %s\n", r.Inputs.DDLSource)
	fmt.Fprintf(&b, "- Comments source: %s\n", r.Inputs.CommentsSource)
	if r.Inputs.PackageID != "" {
		fmt.Fprintf(&b, "- Package: `%s`, version `%s` (previous `%s`, status %s)\n",
			r.Inputs.PackageID, r.Inputs.Version, r.Inputs.PreviousVersion, r.Inputs.Status)
	}
	if r.Inputs.DryRun {
		fmt.Fprintf(&b, "- Mode: dry-run (no APIHUB calls)\n")
	}
	if r.Fatal != nil {
		fmt.Fprintf(&b, "\n## FATAL\n\n`%s`: %s\n", r.Fatal.Code, r.Fatal.Msg)
	}

	st := r.Stats
	fmt.Fprintf(&b, "\n## Stats\n\n")
	fmt.Fprintf(&b, "| Metric | Value |\n|---|---|\n")
	fmt.Fprintf(&b, "| DDL files | %d |\n", st.DdlFiles)
	fmt.Fprintf(&b, "| Tables in DDL / workbook | %d / %d |\n", st.TablesDDL, st.TablesXlsx)
	fmt.Fprintf(&b, "| Columns in DDL / workbook | %d / %d |\n", st.ColumnsDDL, st.ColumnsXlsx)
	fmt.Fprintf(&b, "| Matched tables / columns | %d / %d |\n", st.TablesMatched, st.ColumnsMatched)
	fmt.Fprintf(&b, "| Comments generated (table / column) | %d / %d |\n", st.TableCommentsGenerated, st.ColumnCommentsGenerated)
	fmt.Fprintf(&b, "| FKs generated / unresolved / ambiguous | %d / %d / %d |\n", st.FKsGenerated, st.FKsUnresolved, st.FKsAmbiguous)
	fmt.Fprintf(&b, "| Tables with PK conflicts | %d |\n", st.PKMismatchTables)
	fmt.Fprintf(&b, "| Duplicate rows (exact / conflicting) | %d / %d |\n", st.DupExactRows, st.DupConflicts)
	fmt.Fprintf(&b, "| Empty descriptions (table / column) | %d / %d |\n", st.EmptyTableDesc, st.EmptyColumnDesc)
	fmt.Fprintf(&b, "| Warnings total | %d |\n", len(r.Warnings))

	if r.Publish != nil {
		fmt.Fprintf(&b, "\n## Publish\n\n- publishId: `%s`\n- status: %s\n", r.Publish.PublishID, r.Publish.Status)
		if r.Publish.Message != "" {
			fmt.Fprintf(&b, "- message: %s\n", r.Publish.Message)
		}
		if r.Publish.DurationSec > 0 {
			fmt.Fprintf(&b, "- duration: %.1fs\n", r.Publish.DurationSec)
		}
	}
	if r.Groups != nil {
		fmt.Fprintf(&b, "\n## DDL table groups\n\n")
		switch {
		case r.Groups.Skipped:
			fmt.Fprintf(&b, "Skipped (--skip-groups or disabled in config).\n")
		case !r.Groups.APIAvailable:
			fmt.Fprintf(&b, "The target APIHUB backend does not expose /ddl/groups yet — step skipped.\n")
		default:
			fmt.Fprintf(&b, "- created: %s\n- updated: %s\n- failed: %s\n",
				listOrDash(r.Groups.Created), listOrDash(r.Groups.Updated), listOrDash(r.Groups.Failed))
		}
	}
	if r.Exports != nil {
		fmt.Fprintf(&b, "\n## Exports\n\n")
		for name, e := range map[string]*ExportFile{"entities": r.Exports.Entities, "changes": r.Exports.Changes} {
			if e == nil {
				continue
			}
			if !e.Enriched {
				fmt.Fprintf(&b, "- %s: `%s` (raw export, no custom columns)\n", name, e.Path)
				continue
			}
			fmt.Fprintf(&b, "- %s: `%s` (%d rows, Group filled in %d", name, e.Path, e.Rows, e.GroupFilled)
			if e.GroupAppended {
				fmt.Fprintf(&b, ", Group column appended by the tool")
			}
			if e.AnalyticsFallback {
				fmt.Fprintf(&b, ", Analytics Severity from count columns (fallback)")
			}
			fmt.Fprintf(&b, ")\n")
		}
	}

	order, byCode := r.byCategory()
	if len(order) > 0 {
		fmt.Fprintf(&b, "\n## Warnings\n")
		for _, code := range order {
			ws := byCode[code]
			fmt.Fprintf(&b, "\n### %s (%d)\n\n", code, len(ws))
			fmt.Fprintf(&b, "| Table | Column | Sheet/File | Row | Message |\n|---|---|---|---|---|\n")
			for _, w := range ws {
				loc := w.Sheet
				if loc == "" {
					loc = w.File
				}
				row := ""
				if w.Row > 0 {
					row = fmt.Sprintf("%d", w.Row)
				}
				fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n",
					mdCell(w.Table), mdCell(w.Column), mdCell(loc), row, mdCell(w.Msg))
			}
		}
	}
	return b.String()
}

func mdCell(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	return strings.ReplaceAll(s, "\n", " ")
}

func listOrDash(items []string) string {
	if len(items) == 0 {
		return "—"
	}
	return strings.Join(items, ", ")
}
