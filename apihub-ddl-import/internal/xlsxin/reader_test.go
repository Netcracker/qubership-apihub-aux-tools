package xlsxin

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"apihub-ddl-import/internal/model"
)

func readCase(t *testing.T, name string) ([]byte, error) {
	t.Helper()
	return os.ReadFile(fmt.Sprintf("../../testdata/cases/%s/comments-and-pfk.xlsx", name))
}

func TestReadBaseline(t *testing.T) {
	data, err := readCase(t, "case-01")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := Read(data)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(doc.Tables) != 75 {
		t.Errorf("table rows = %d, want 75", len(doc.Tables))
	}
	if len(doc.Columns) != 1081 {
		t.Errorf("column rows = %d, want 1081", len(doc.Columns))
	}
	if len(doc.Warnings) != 0 {
		t.Errorf("warnings = %v, want none", doc.Warnings)
	}
	tr := doc.Tables[0]
	if tr.Domain == "" || tr.Table == "" || tr.Row < 2 {
		t.Errorf("first table row looks wrong: %+v", tr)
	}
	cr := doc.Columns[0]
	if cr.Table == "" || cr.Column == "" || cr.Row < 2 {
		t.Errorf("first column row looks wrong: %+v", cr)
	}
}

func TestReadAllPublishableCases(t *testing.T) {
	for _, c := range []string{"case-01", "case-02", "case-03", "case-04", "case-05", "case-06", "case-07", "case-10"} {
		data, err := readCase(t, c)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Read(data); err != nil {
			t.Errorf("%s: unexpected error: %v", c, err)
		}
	}
}

func TestReadFatalEmpty(t *testing.T) {
	data, err := readCase(t, "case-08")
	if err != nil {
		t.Fatal(err)
	}
	_, err = Read(data)
	var fe *model.FatalError
	if !errors.As(err, &fe) || fe.Code != model.FXlsxEmpty {
		t.Fatalf("err = %v, want FatalError %s", err, model.FXlsxEmpty)
	}
}

func TestReadFatalBrokenHeader(t *testing.T) {
	data, err := readCase(t, "case-09")
	if err != nil {
		t.Fatal(err)
	}
	_, err = Read(data)
	var fe *model.FatalError
	if !errors.As(err, &fe) || fe.Code != model.FXlsxHeader {
		t.Fatalf("err = %v, want FatalError %s", err, model.FXlsxHeader)
	}
}
