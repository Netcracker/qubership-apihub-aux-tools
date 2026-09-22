// Package model holds the shared data types of the DDL import pipeline:
// parsed DDL, parsed comments workbook, merge results, warnings and fatal errors.
package model

import "fmt"

// Warning codes — stable identifiers used in report.json and asserted by tests.
const (
	WTblDdlOnly      = "TBL_DDL_ONLY"      // table present in DDL, absent in the workbook
	WTblXlsxOnly     = "TBL_XLSX_ONLY"     // table present in the workbook, absent in DDL
	WColDdlOnly      = "COL_DDL_ONLY"      // column present in DDL, absent in the workbook
	WColXlsxOnly     = "COL_XLSX_ONLY"     // column present in the workbook, absent in DDL
	WDescEmptyTable  = "DESC_EMPTY_TABLE"  // matched table with empty description
	WDescEmptyColumn = "DESC_EMPTY_COLUMN" // matched column with empty description
	WPkMismatch      = "PK_MISMATCH"       // workbook PK marks contradict the DDL primary key (DDL wins)
	WFkUnresolved    = "FK_UNRESOLVED"     // FK/PFK mark with no target resolvable by the <table_stem>_id convention
	WFkAmbiguous     = "FK_AMBIGUOUS"      // FK/PFK mark with several convention candidates after tie-breaks
	WFkExists        = "FK_EXISTS"         // FK already declared in the source DDL, generation skipped
	WPfkWithoutPk    = "PFK_WITHOUT_PK"    // Constraint=PFK on a row without IsPK=Y
	WDupExact        = "DUP_EXACT"         // duplicate workbook row (identical after normalization), dropped
	WDupConflict     = "DUP_CONFLICT"      // duplicate workbook rows with conflicting content, no comment emitted
	WBadFlag         = "BAD_FLAG"          // unrecognized IsPK/Constraint value
	WSheetEmpty      = "SHEET_EMPTY"       // one sheet has no data rows (both empty is fatal)
	WCommentExists   = "COMMENT_EXISTS"    // source DDL already carries COMMENT ON for this target (ours appended, last wins)
	WDupDdlTable     = "DUP_DDL_TABLE"     // same table defined twice in the DDL sources (first wins)

	WGroupsApiUnavailable = "GROUPS_API_UNAVAILABLE"    // backend has no /ddl/groups endpoints
	WGroupCreateFailed    = "GROUP_CREATE_FAILED"       // group creation/update failed for one domain
	WExportUnavailable    = "EXPORT_UNAVAILABLE"        // an xlsx export endpoint returned an error
	WExportLayoutUnknown  = "EXPORT_LAYOUT_UNKNOWN"     // export sheet/headers not recognized, file kept as-is
	WExportGroupMismatch  = "EXPORT_GROUP_MISMATCH"     // backend Group cell disagrees with the workbook domain
	WAnalyticsFallback    = "ANALYTICS_FALLBACK_COUNTS" // Analytics Severity derived from count columns, not per-change data
	WDescriptionsMissing  = "DESCRIPTIONS_MISSING"      // published DDL entities have no descriptions (builder ignored COMMENT ON?)
)

// Fatal error codes — pipeline aborts, nothing is published.
const (
	FXlsxHeader          = "F_XLSX_HEADER"            // workbook sheets/headers do not match the contract
	FXlsxEmpty           = "F_XLSX_EMPTY"             // both sheets are header-only
	FNoDdlFiles          = "F_NO_DDL_FILES"           // DDL source yielded no .sql files
	FDdlParse            = "F_DDL_PARSE"              // a DDL file failed to parse
	FPrevVersionNotFound = "F_PREV_VERSION_NOT_FOUND" // --previous-version does not exist in APIHUB
	FPublishFailed       = "F_PUBLISH_FAILED"         // publish build ended in error or timed out
	FApihubUnreachable   = "F_APIHUB_UNREACHABLE"     // APIHUB preflight request failed
	FNoDdlEntities       = "F_NO_DDL_ENTITIES"        // published version has zero DDL entities
	FGroupsFailed        = "F_GROUPS_FAILED"          // DDL table groups were attempted but not a single one succeeded
)

// FatalError aborts the pipeline (exit code 1).
type FatalError struct {
	Code string
	Msg  string
}

func (e *FatalError) Error() string { return fmt.Sprintf("%s: %s", e.Code, e.Msg) }

// Fatalf builds a FatalError with a formatted message.
func Fatalf(code, format string, args ...any) *FatalError {
	return &FatalError{Code: code, Msg: fmt.Sprintf(format, args...)}
}

// Warning is a single non-fatal finding, attributable to a table/column and,
// for workbook findings, to a 1-based row of the source sheet.
type Warning struct {
	Code   string `json:"code"`
	Msg    string `json:"msg"`
	Table  string `json:"table,omitempty"`
	Column string `json:"column,omitempty"`
	File   string `json:"file,omitempty"`
	Sheet  string `json:"sheet,omitempty"`
	Row    int    `json:"row,omitempty"`
}

// File is one input source file (path relative to the source root, forward slashes).
type File struct {
	RelPath string
	Data    []byte
}

// DDLColumn is a column of a parsed CREATE TABLE.
type DDLColumn struct {
	Name    string
	Type    string
	NotNull bool
	Pos     int // 0-based position within the table
}

// DDLTable is one CREATE TABLE statement of the source DDL.
type DDLTable struct {
	Schema        string // "" when unqualified
	Name          string
	QualifiedName string // schema.name as it should appear in generated statements
	File          string // RelPath of the defining file
	Columns       []DDLColumn
	PKName        string
	PKCols        []string
	// ConstraintNames lists all constraint names declared on the table (PK/FK/UNIQUE/CHECK)
	// so generated FK names can avoid collisions.
	ConstraintNames []string
	// ExistingFKCols are local columns already covered by a FOREIGN KEY in the source DDL.
	ExistingFKCols []string
	// InsertOffset is the byte offset in the (marker-stripped) file content where the
	// per-table enrichment block is spliced: the start of the line following the
	// statement's terminating semicolon.
	InsertOffset int
	StmtStart    int
	StmtEnd      int
}

// Column returns the column with the given (exact) name, or nil.
func (t *DDLTable) Column(name string) *DDLColumn {
	for i := range t.Columns {
		if t.Columns[i].Name == name {
			return &t.Columns[i]
		}
	}
	return nil
}

// DDLFile is one parsed source file; Content is the original text with any
// previously generated apihub-ddl-import marker blocks stripped.
type DDLFile struct {
	RelPath string
	Content string
	Tables  []*DDLTable // in definition order
}

// DDLModel is the parsed DDL source set.
type DDLModel struct {
	Files  []*DDLFile
	Tables []*DDLTable // all tables in definition order across files
	// ExistingTableComments / ExistingColumnComments record COMMENT ON targets already
	// present in the sources, keyed by normalized table name (and "table|column").
	ExistingTableComments  map[string]bool
	ExistingColumnComments map[string]bool
}

// TableRow is one data row of the "List of Tables" sheet.
type TableRow struct {
	Domain   string
	Table    string
	Desc     string
	External string
	Row      int // 1-based row in the sheet
}

// ColumnRow is one data row of the "Tables Specifications" sheet.
type ColumnRow struct {
	Domain        string
	Table         string
	Column        string
	DataType      string
	IsPK          string // raw trimmed sheet value; canonicalized to "Y"/"" during merge
	Constraint    string // raw trimmed sheet value; canonicalized to PK/FK/PFK/"" during merge
	Desc          string
	DeployRelease string
	RDB           string
	ExternalDB    string
	Row           int // 1-based row in the sheet
}

// CommentsDoc is the parsed comments workbook.
type CommentsDoc struct {
	Tables   []TableRow
	Columns  []ColumnRow
	Warnings []Warning // reader-level findings (SHEET_EMPTY, BAD_FLAG, ...)
}

// ResolvedFK is a foreign key resolved by the <table_stem>_id naming convention.
type ResolvedFK struct {
	File         string // file of the referencing table (FK block goes to its end)
	Table        string // referencing table (DDL name)
	Column       string // referencing column (DDL name)
	TargetTable  string
	TargetColumn string
	Name         string // generated constraint name
	Note         string // e.g. composite-PK target explanation
}

// Stats are the merge counters shown in the report and asserted by fixture tests.
type Stats struct {
	DdlFiles                int `json:"ddlFiles"`
	TablesDDL               int `json:"tablesDdl"`
	TablesXlsx              int `json:"tablesXlsx"`
	ColumnsDDL              int `json:"columnsDdl"`
	ColumnsXlsx             int `json:"columnsXlsx"`
	TablesMatched           int `json:"tablesMatched"`
	ColumnsMatched          int `json:"columnsMatched"`
	TableCommentsGenerated  int `json:"tableCommentsGenerated"`
	ColumnCommentsGenerated int `json:"columnCommentsGenerated"`
	FKsGenerated            int `json:"fksGenerated"`
	FKsUnresolved           int `json:"fksUnresolved"`
	FKsAmbiguous            int `json:"fksAmbiguous"`
	PKMismatchTables        int `json:"pkMismatchTables"`
	DupExactRows            int `json:"dupExactRows"`
	DupConflicts            int `json:"dupConflicts"`
	EmptyTableDesc          int `json:"emptyTableDesc"`
	EmptyColumnDesc         int `json:"emptyColumnDesc"`
	WarningsTotal           int `json:"warningsTotal"`
}

// MergeResult is the outcome of merging the workbook into the DDL model.
type MergeResult struct {
	// TableComments maps DDL table name → comment text (unescaped).
	TableComments map[string]string
	// ColumnComments maps DDL table name → DDL column name → comment text.
	ColumnComments map[string]map[string]string
	FKs            []ResolvedFK
	Warnings       []Warning
	Stats          Stats
	// DomainByTable maps normalized table name → workbook Domain.
	DomainByTable map[string]string
	// Domains lists unique domains in first-seen sheet order.
	Domains []string
}

// CountWarnings returns the number of warnings with the given code.
func CountWarnings(ws []Warning, code string) int {
	n := 0
	for _, w := range ws {
		if w.Code == code {
			n++
		}
	}
	return n
}
