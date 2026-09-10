package pipeline

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"apihub-ddl-import/internal/config"
)

func testOptions(t *testing.T, caseName string, strict bool) Options {
	t.Helper()
	cfg := &config.Config{}
	cfg.DDLSource = config.Source{Type: config.SourceFile, Path: "../../testdata/ddl"}
	cfg.CommentsSource = config.Source{Type: config.SourceFile,
		Path: filepath.Join("../../testdata/cases", caseName, "comments-and-pfk.xlsx")}
	cfg.Output.Dir = t.TempDir()
	cfg.ApplyDefaults()
	return Options{
		Cfg:             cfg,
		Version:         "2026.1",
		PreviousVersion: PreviousVersionNone,
		Status:          "draft",
		DryRun:          true,
		Strict:          strict,
		PublishTimeout:  time.Minute,
	}
}

func TestDryRunCase01(t *testing.T) {
	opts := testOptions(t, "case-01", false)
	if code := Run(opts); code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}
	for _, p := range []string{
		"enriched/base.sql", "report.md", "report.json", "raw/ddl/base.sql", "raw/comments-and-pfk.xlsx",
	} {
		if _, err := os.Stat(filepath.Join(opts.Cfg.Output.Dir, p)); err != nil {
			t.Errorf("artifact %s missing: %v", p, err)
		}
	}
}

func TestDryRunFatalCases(t *testing.T) {
	for _, c := range []string{"case-08", "case-09"} {
		opts := testOptions(t, c, false)
		if code := Run(opts); code != ExitFatal {
			t.Errorf("%s: exit = %d, want %d", c, code, ExitFatal)
		}
		if _, err := os.Stat(filepath.Join(opts.Cfg.Output.Dir, "enriched")); !os.IsNotExist(err) {
			t.Errorf("%s: enriched output must not exist on fatal", c)
		}
		// The failure report is still written.
		if _, err := os.Stat(filepath.Join(opts.Cfg.Output.Dir, "report.json")); err != nil {
			t.Errorf("%s: report.json missing: %v", c, err)
		}
	}
}

func TestDryRunStrictStopsOnWarnings(t *testing.T) {
	opts := testOptions(t, "case-10", true)
	if code := Run(opts); code != ExitStrict {
		t.Fatalf("exit = %d, want %d", code, ExitStrict)
	}
}
