package pipeline

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	"apihub-ddl-import/internal/config"
	"apihub-ddl-import/internal/model"
)

// buildXlsx assembles a minimal single-sheet "DDL" workbook (the shape APIHUB's
// export endpoints return) with the given header row and data rows.
func buildXlsx(t *testing.T, headers []string, rows [][]any) []byte {
	t.Helper()
	f := excelize.NewFile()
	const sheet = "DDL"
	if _, err := f.NewSheet(sheet); err != nil {
		t.Fatal(err)
	}
	if err := f.DeleteSheet("Sheet1"); err != nil {
		t.Fatal(err)
	}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := f.SetCellValue(sheet, cell, h); err != nil {
			t.Fatal(err)
		}
	}
	for r, row := range rows {
		for c, v := range row {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+2)
			if err := f.SetCellValue(sheet, cell, v); err != nil {
				t.Fatal(err)
			}
		}
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	return buf.Bytes()
}

// buildInputCommentsXlsx writes a comments workbook satisfying the xlsxin
// contract (sheets "List of Tables" / "Tables Specifications") describing the
// single table "t1" defined by writeMiniDDL, and returns its path.
func buildInputCommentsXlsx(t *testing.T) string {
	t.Helper()
	f := excelize.NewFile()
	if _, err := f.NewSheet("List of Tables"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.NewSheet("Tables Specifications"); err != nil {
		t.Fatal(err)
	}
	if err := f.DeleteSheet("Sheet1"); err != nil {
		t.Fatal(err)
	}
	setRow := func(sheet string, row int, vals ...any) {
		for c, v := range vals {
			cell, _ := excelize.CoordinatesToCellName(c+1, row)
			if err := f.SetCellValue(sheet, cell, v); err != nil {
				t.Fatal(err)
			}
		}
	}
	setRow("List of Tables", 1, "Domain", "Table Name", "Table Description", "External Table")
	setRow("List of Tables", 2, "DOM", "t1", "Table one", "N")
	setRow("Tables Specifications", 1, "Domain", "Table Name", "Column Name", "Data Type", "IsPK",
		"Constraint", "Column Description", "Deployment Release", "RDB", "ExternalDB")
	setRow("Tables Specifications", 2, "DOM", "t1", "id", "text", "Y", "PK", "Identifier", "1", "Y", "N")
	setRow("Tables Specifications", 3, "DOM", "t1", "name", "text", "", "", "Name of the thing", "1", "Y", "N")

	path := filepath.Join(t.TempDir(), "comments-and-pfk.xlsx")
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return path
}

func writeMiniDDL(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "base.sql")
	sql := "CREATE TABLE t1 (id text NOT NULL, name text, CONSTRAINT t1_pk PRIMARY KEY (id));\n"
	if err := os.WriteFile(path, []byte(sql), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

var entitiesExportHeaders = []string{"Package ID", "Package Name", "Version", "Schema Name", "Name", "Description", "Document ID", "Group"}
var changesExportHeaders = []string{"Version", "Previous Version", "Schema Name", "Name", "Kind",
	"Breaking", "Semi-breaking", "Deprecated", "Non-breaking", "Annotation", "Unclassified", "Group"}

// newFakeApihub starts an httptest server that fakes the slice of the APIHUB
// API a non-dry-run Run() drives: version preflight, a synchronous (204)
// publish, the post-publish entities gate, and the two DDL exports. Table
// groups are out of scope here (the test runs with SkipGroups: true).
func newFakeApihub(t *testing.T, pkg, version, prevVersion string, entitiesXlsx, changesXlsx []byte) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	versionPath := func(v string) string { return "/api/v3/packages/" + pkg + "/versions/" + v }
	mux.HandleFunc(versionPath(version), func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r) // not published yet — this is the version we're about to create
	})
	mux.HandleFunc(versionPath(prevVersion), func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"packageId": pkg, "version": prevVersion, "status": "release"})
	})
	mux.HandleFunc("/api/v2/packages/"+pkg+"/publish", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent) // synchronous accept: nothing to poll
	})
	entitiesBase := "/api/v1/packages/" + pkg + "/versions/" + version + "/ddl/"
	mux.HandleFunc(entitiesBase+"entities", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"entities": []map[string]any{
			{"ddlEntityId": "e1", "kind": "table", "schemaName": "public", "name": "t1", "description": "d1"},
		}})
	})
	mux.HandleFunc(entitiesBase+"export/entities", func(w http.ResponseWriter, r *http.Request) {
		w.Write(entitiesXlsx)
	})
	mux.HandleFunc(entitiesBase+"export/changes", func(w http.ResponseWriter, r *http.Request) {
		w.Write(changesXlsx)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func runPipelineAgainstFake(t *testing.T, srv *httptest.Server, pkg, version, prevVersion, ddlPath, commentsPath string, skipEnrichment bool) string {
	t.Helper()
	outDir := t.TempDir()
	cfg := &config.Config{}
	cfg.Apihub.URL = srv.URL
	cfg.Apihub.APIKey = "test-key"
	cfg.Apihub.PackageID = pkg
	cfg.DDLSource = config.Source{Type: config.SourceFile, Path: ddlPath}
	cfg.CommentsSource = config.Source{Type: config.SourceFile, Path: commentsPath}
	cfg.Output.Dir = outDir
	cfg.ApplyDefaults()
	opts := Options{
		Cfg:             cfg,
		Version:         version,
		PreviousVersion: prevVersion,
		Status:          "draft",
		SkipGroups:      true,
		SkipEnrichment:  skipEnrichment,
		PublishTimeout:  time.Minute,
	}
	if code := Run(opts); code != ExitOK {
		t.Fatalf("skipEnrichment=%v: exit = %d, want %d", skipEnrichment, code, ExitOK)
	}
	return outDir
}

// cellAt returns row[idx], or "" if the row is shorter (excelize trims
// trailing empty cells from GetRows).
func cellAt(row []string, idx int) string {
	if idx >= len(row) {
		return ""
	}
	return row[idx]
}

func readSheetCells(t *testing.T, path string) [][]string {
	t.Helper()
	f, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()
	rows, err := f.GetRows("DDL")
	if err != nil {
		t.Fatalf("read sheet DDL of %s: %v", path, err)
	}
	return rows
}

// TestPublishSkipEnrichmentToggle drives a full (non-dry-run) publish against a
// fake APIHUB and asserts the --skip-enrichment flag controls exactly the
// export enrichment step: raw APIHUB bytes pass through unmodified when set,
// and the usual Group/Analytics Severity columns appear when it's not.
func TestPublishSkipEnrichmentToggle(t *testing.T) {
	const pkg, version, prevVersion = "TEST.PKG", "2026.2", "2026.1"

	entitiesXlsx := buildXlsx(t, entitiesExportHeaders,
		[][]any{{pkg, "Test Pkg", version, "public", "t1", "desc", "doc1", ""}})
	changesXlsx := buildXlsx(t, changesExportHeaders,
		[][]any{{version, prevVersion, "public", "t1", "table", 0, 0, 0, 1, 0, 0, ""}})

	ddlPath := writeMiniDDL(t)
	commentsPath := buildInputCommentsXlsx(t)

	// --- enrichment ON (default) ---
	srv := newFakeApihub(t, pkg, version, prevVersion, entitiesXlsx, changesXlsx)
	outDir := runPipelineAgainstFake(t, srv, pkg, version, prevVersion, ddlPath, commentsPath, false)

	entPath := filepath.Join(outDir, "export", "DDLEntities_"+pkg+"_"+version+".xlsx")
	entRows := readSheetCells(t, entPath)
	if entRows[0][7] != "Group" || entRows[1][7] != "DOM" {
		t.Errorf("enriched entities export: header=%v row=%v, want Group=DOM filled from the workbook domain", entRows[0], entRows[1])
	}

	chgPath := filepath.Join(outDir, "export", "DDLChanges_"+pkg+"_"+version+".xlsx")
	chgRows := readSheetCells(t, chgPath)
	if len(chgRows[0]) < 13 || chgRows[0][12] != "Analytics Severity" {
		t.Fatalf("enriched changes export header = %v, want Analytics Severity appended", chgRows[0])
	}
	if chgRows[1][11] != "DOM" || chgRows[1][12] != "non-breaking" {
		t.Errorf("enriched changes export row = %v, want Group=DOM, Analytics Severity=non-breaking (fallback from Non-breaking=1)", chgRows[1])
	}

	reportData, err := os.ReadFile(filepath.Join(outDir, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(reportData, []byte(`"enriched": true`)) {
		t.Error(`report.json must record exports.entities.enriched=true when enrichment ran`)
	}

	// --- enrichment OFF (--skip-enrichment) ---
	srv2 := newFakeApihub(t, pkg, version, prevVersion, entitiesXlsx, changesXlsx)
	outDir2 := runPipelineAgainstFake(t, srv2, pkg, version, prevVersion, ddlPath, commentsPath, true)

	rawEntPath := filepath.Join(outDir2, "export", "DDLEntities_"+pkg+"_"+version+".xlsx")
	gotRaw, err := os.ReadFile(rawEntPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotRaw, entitiesXlsx) {
		t.Error("--skip-enrichment: entities export must be byte-identical to APIHUB's raw response")
	}
	rawEntRows := readSheetCells(t, rawEntPath)
	if got := cellAt(rawEntRows[1], 7); got != "" {
		t.Errorf("--skip-enrichment: Group cell must stay as APIHUB returned it (empty), got %q", got)
	}

	rawChgPath := filepath.Join(outDir2, "export", "DDLChanges_"+pkg+"_"+version+".xlsx")
	gotRawChg, err := os.ReadFile(rawChgPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotRawChg, changesXlsx) {
		t.Error("--skip-enrichment: changes export must be byte-identical to APIHUB's raw response")
	}
	rawChgRows := readSheetCells(t, rawChgPath)
	if len(rawChgRows[0]) >= 13 {
		t.Errorf("--skip-enrichment: changes export must NOT gain an Analytics Severity column, header = %v", rawChgRows[0])
	}

	reportData2, err := os.ReadFile(filepath.Join(outDir2, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(reportData2, []byte(`"enriched": false`)) {
		t.Error(`report.json must record exports.entities.enriched=false under --skip-enrichment`)
	}
	if !bytes.Contains(reportData2, []byte("raw export, no custom columns")) {
		// report.md is written alongside report.json; check that too.
		mdData, _ := os.ReadFile(filepath.Join(outDir2, "report.md"))
		if !bytes.Contains(mdData, []byte("raw export, no custom columns")) {
			t.Error("report.md must note the raw/skipped-enrichment exports")
		}
	}
}

// TestPublishGroupsTotalFailureIsFatal locks in a live-discovered requirement:
// a published version with zero successfully created DDL table groups is a
// failed deliverable, not a warning to bury under an otherwise-green exit
// code — this applies whether the backend flatly lacks /ddl/groups (observed
// live: APIHUB answering 421 "Requested unknown endpoint" for every domain)
// or every individual domain attempt fails for some other reason.
func TestPublishGroupsTotalFailureIsFatal(t *testing.T) {
	const pkg, version, prevVersion = "TEST.PKG", "2026.2", "2026.1"

	ddlPath := writeMiniDDL(t)
	commentsPath := buildInputCommentsXlsx(t)

	mux := http.NewServeMux()
	versionPath := func(v string) string { return "/api/v3/packages/" + pkg + "/versions/" + v }
	mux.HandleFunc(versionPath(version), func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) })
	mux.HandleFunc(versionPath(prevVersion), func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"packageId": pkg, "version": prevVersion, "status": "release"})
	})
	mux.HandleFunc("/api/v2/packages/"+pkg+"/publish", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	ddlBase := "/api/v1/packages/" + pkg + "/versions/" + version + "/ddl/"
	mux.HandleFunc(ddlBase+"entities", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"entities": []map[string]any{
			{"ddlEntityId": "e1", "kind": "table", "schemaName": "public", "name": "t1", "description": "d1"},
		}})
	})
	mux.HandleFunc(ddlBase+"groups", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusMisdirectedRequest)
		w.Write([]byte(`{"status":421,"message":"Requested unknown endpoint"}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	outDir := t.TempDir()
	cfg := &config.Config{}
	cfg.Apihub.URL = srv.URL
	cfg.Apihub.APIKey = "test-key"
	cfg.Apihub.PackageID = pkg
	cfg.DDLSource = config.Source{Type: config.SourceFile, Path: ddlPath}
	cfg.CommentsSource = config.Source{Type: config.SourceFile, Path: commentsPath}
	cfg.Output.Dir = outDir
	cfg.ApplyDefaults()
	opts := Options{
		Cfg:             cfg,
		Version:         version,
		PreviousVersion: prevVersion,
		Status:          "draft",
		SkipExports:     true, // irrelevant to this test, and the fatal exit happens before exports anyway
		PublishTimeout:  time.Minute,
	}
	if code := Run(opts); code != ExitFatal {
		t.Fatalf("exit = %d, want %d (ExitFatal)", code, ExitFatal)
	}
	reportData, err := os.ReadFile(filepath.Join(outDir, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(reportData, []byte(model.FGroupsFailed)) {
		t.Errorf("report.json must record the %s fatal code", model.FGroupsFailed)
	}
}
