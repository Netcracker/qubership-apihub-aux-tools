package gitlab

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewParsesRepoURL(t *testing.T) {
	c, err := New("https://gitlab.example.com/group/sub/repo.git/", "tok", false)
	if err != nil {
		t.Fatal(err)
	}
	if c.APIBase != "https://gitlab.example.com/api/v4" {
		t.Errorf("api base = %q", c.APIBase)
	}
	if c.Project != "group%2Fsub%2Frepo" {
		t.Errorf("project = %q, want group%%2Fsub%%2Frepo (QueryEscape, not PathEscape)", c.Project)
	}
	if _, err := New("gitlab.com/no-scheme", "", false); err == nil {
		t.Error("relative URL must be rejected")
	}
	if _, err := New("https://gitlab.com/", "", false); err == nil {
		t.Error("URL without project path must be rejected")
	}
}

func TestListTreePaginationAndAuth(t *testing.T) {
	var gotToken string
	var pages []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.Header.Get("PRIVATE-TOKEN")
		if !strings.HasPrefix(r.URL.Path, "/api/v4/projects/grp%2Frepo/repository/tree") &&
			!strings.HasPrefix(r.URL.RawPath, "/api/v4/projects/grp%2Frepo/repository/tree") {
			t.Errorf("unexpected path %q raw %q", r.URL.Path, r.URL.RawPath)
		}
		page := r.URL.Query().Get("page")
		pages = append(pages, page)
		w.Header().Set("Content-Type", "application/json")
		switch page {
		case "1":
			w.Header().Set("X-Next-Page", "2")
			json.NewEncoder(w).Encode([]TreeEntry{{Name: "a.sql", Type: "blob", Path: "db/a.sql"}})
		default:
			w.Header().Set("X-Next-Page", "")
			json.NewEncoder(w).Encode([]TreeEntry{{Name: "b.sql", Type: "blob", Path: "db/b.sql"}})
		}
	}))
	defer srv.Close()

	c := &Client{APIBase: srv.URL + "/api/v4", Project: "grp%2Frepo", Token: "secret", HTTPClient: srv.Client()}
	entries, err := c.ListTree("db", "main")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].Path != "db/a.sql" || entries[1].Path != "db/b.sql" {
		t.Errorf("entries = %+v", entries)
	}
	if gotToken != "secret" {
		t.Errorf("PRIVATE-TOKEN = %q", gotToken)
	}
	if len(pages) != 2 {
		t.Errorf("pages requested: %v, want 2 pages", pages)
	}
}

func TestGetRawFileEncodesPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The file path must arrive URL-encoded as a single segment.
		want := "/api/v4/projects/grp%2Frepo/repository/files/db%2Fsub%2Ffile.sql/raw"
		got := r.URL.EscapedPath()
		if got != want {
			t.Errorf("path = %q, want %q", got, want)
		}
		if r.URL.Query().Get("ref") != "feature/x" {
			t.Errorf("ref = %q", r.URL.Query().Get("ref"))
		}
		w.Write([]byte("CREATE TABLE t();"))
	}))
	defer srv.Close()

	c := &Client{APIBase: srv.URL + "/api/v4", Project: "grp%2Frepo", HTTPClient: srv.Client()}
	data, err := c.GetRawFile("db/sub/file.sql", "feature/x")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "CREATE TABLE t();" {
		t.Errorf("data = %q", data)
	}
}

func TestGetRawFileError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"404 File Not Found"}`, http.StatusNotFound)
	}))
	defer srv.Close()
	c := &Client{APIBase: srv.URL + "/api/v4", Project: "p", HTTPClient: srv.Client()}
	if _, err := c.GetRawFile("nope.sql", "main"); err == nil || !strings.Contains(err.Error(), "404") {
		t.Errorf("err = %v, want 404 mention", err)
	}
}
