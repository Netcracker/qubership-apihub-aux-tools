package apihub

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestClient(srv *httptest.Server) *Client {
	c := New(srv.URL, "test-key", false)
	c.HTTPClient = srv.Client()
	return c
}

func TestGetVersion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(APIKeyHeader) != "test-key" {
			t.Errorf("api-key header = %q", r.Header.Get(APIKeyHeader))
		}
		switch r.URL.Path {
		case "/api/v3/packages/pkg.a/versions/2026.1":
			json.NewEncoder(w).Encode(VersionInfo{PackageID: "pkg.a", Version: "2026.1", Status: "release"})
		default:
			http.Error(w, `{"code":"2100","message":"version not found"}`, http.StatusNotFound)
		}
	}))
	defer srv.Close()
	c := newTestClient(srv)

	v, found, err := c.GetVersion("pkg.a", "2026.1")
	if err != nil || !found || v.Status != "release" {
		t.Errorf("got %+v found=%v err=%v", v, found, err)
	}
	_, found, err = c.GetVersion("pkg.a", "missing")
	if err != nil || found {
		t.Errorf("missing version: found=%v err=%v", found, err)
	}
}

func TestPublishVersionMultipart(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/packages/pkg.a/publish" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			t.Fatalf("parse multipart: %v", err)
		}
		cfg := r.FormValue("config")
		if !strings.Contains(cfg, `"version":"2026.1"`) {
			t.Errorf("config field = %q", cfg)
		}
		f, hdr, err := r.FormFile("sources")
		if err != nil {
			t.Fatalf("sources part: %v", err)
		}
		defer f.Close()
		if hdr.Filename != "sources.zip" {
			t.Errorf("sources filename = %q", hdr.Filename)
		}
		data, _ := io.ReadAll(f)
		if string(data) != "ZIPBYTES" {
			t.Errorf("sources data = %q", data)
		}
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprint(w, `{"publishId":"pub-1"}`)
	}))
	defer srv.Close()
	c := newTestClient(srv)

	id, sync, err := c.PublishVersion("pkg.a", []byte("ZIPBYTES"), []byte(`{"version":"2026.1"}`))
	if err != nil || sync || id != "pub-1" {
		t.Errorf("id=%q sync=%v err=%v", id, sync, err)
	}
}

func TestGetPublishStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/packages/pkg.a/publish/pub-1/status" {
			t.Errorf("path = %q", r.URL.Path)
		}
		fmt.Fprint(w, `{"publishId":"pub-1","status":"complete","message":""}`)
	}))
	defer srv.Close()
	st, err := newTestClient(srv).GetPublishStatus("pkg.a", "pub-1")
	if err != nil || st.Status != "complete" {
		t.Errorf("st=%+v err=%v", st, err)
	}
}

func TestListDdlEntitiesPagination(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/packages/pkg.a/versions/2026.1/ddl/entities" {
			t.Errorf("path = %q", r.URL.Path)
		}
		offset := r.URL.Query().Get("offset")
		w.Header().Set("Content-Type", "application/json")
		if offset == "0" {
			// full page of 100 → client must request the next one
			ents := make([]DdlEntity, 100)
			for i := range ents {
				ents[i] = DdlEntity{DdlEntityId: fmt.Sprintf("tbl-%d", i), Name: fmt.Sprintf("t%d", i)}
			}
			json.NewEncoder(w).Encode(map[string]any{"entities": ents})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"entities": []DdlEntity{{DdlEntityId: "tbl-last", Name: "tlast"}}})
	}))
	defer srv.Close()
	ents, err := newTestClient(srv).ListDdlEntities("pkg.a", "2026.1")
	if err != nil {
		t.Fatal(err)
	}
	if len(ents) != 101 || ents[100].DdlEntityId != "tbl-last" {
		t.Errorf("entities = %d, want 101", len(ents))
	}
}

func TestCreateDdlGroupErrors(t *testing.T) {
	var mode string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch mode {
		case "404":
			http.NotFound(w, r)
		case "dup":
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `{"status":400,"code":"8403","message":"DDL table group with groupName=ACM already exists"}`)
		default:
			var req CreateDdlGroupReq
			json.NewDecoder(r.Body).Decode(&req)
			if req.GroupName != "ACM" || len(req.Tables) != 2 || req.Tables[0].DdlEntityId != "tbl-1" {
				t.Errorf("req = %+v", req)
			}
			w.WriteHeader(http.StatusCreated)
		}
	}))
	defer srv.Close()
	c := newTestClient(srv)
	body := CreateDdlGroupReq{GroupName: "ACM", Tables: []DdlGroupTable{{DdlEntityId: "tbl-1"}, {DdlEntityId: "tbl-2"}}}

	mode = "ok"
	if err := c.CreateDdlGroup("pkg.a", "2026.1", body); err != nil {
		t.Errorf("create: %v", err)
	}
	mode = "404"
	err := c.CreateDdlGroup("pkg.a", "2026.1", body)
	var ae *APIError
	if !errors.As(err, &ae) || ae.HTTPStatus != http.StatusNotFound {
		t.Errorf("404 err = %v", err)
	}
	mode = "dup"
	err = c.CreateDdlGroup("pkg.a", "2026.1", body)
	if !errors.As(err, &ae) || ae.Code != "8403" {
		t.Errorf("dup err = %v (code %q)", err, ae.Code)
	}
}

func TestUpdateDdlGroup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/api/v1/packages/pkg.a/versions/2026.1/ddl/groups/ACM" {
			t.Errorf("%s %s", r.Method, r.URL.Path)
		}
		var req UpdateDdlGroupReq
		json.NewDecoder(r.Body).Decode(&req)
		if req.Tables == nil || len(*req.Tables) != 1 {
			t.Errorf("tables = %+v", req.Tables)
		}
		if req.GroupName != nil {
			t.Error("groupName must be omitted when not renaming")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	desc := "d"
	tables := []DdlGroupTable{{DdlEntityId: "tbl-1"}}
	err := newTestClient(srv).UpdateDdlGroup("pkg.a", "2026.1", "ACM", UpdateDdlGroupReq{Description: &desc, Tables: &tables})
	if err != nil {
		t.Fatal(err)
	}
}

func TestExportDdlChangesParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/packages/pkg.a/versions/2026.2/ddl/export/changes" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("previousVersion") != "2026.1" {
			t.Errorf("previousVersion = %q", r.URL.Query().Get("previousVersion"))
		}
		w.Write([]byte("XLSX"))
	}))
	defer srv.Close()
	data, err := newTestClient(srv).ExportDdlChanges("pkg.a", "2026.2", "2026.1", "")
	if err != nil || string(data) != "XLSX" {
		t.Errorf("data=%q err=%v", data, err)
	}
}

func TestGetDdlEntityChanges(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/packages/pkg.a/versions/2026.2/ddl/entities/tbl-1/changes" {
			t.Errorf("path = %q", r.URL.Path)
		}
		fmt.Fprint(w, `{"changes":[{"description":"Column added","severity":"breaking"},{"description":"Comment updated","severity":"annotation"}]}`)
	}))
	defer srv.Close()
	chs, err := newTestClient(srv).GetDdlEntityChanges("pkg.a", "2026.2", "tbl-1", "2026.1", "")
	if err != nil || len(chs) != 2 || chs[1].Severity != "annotation" {
		t.Errorf("chs=%+v err=%v", chs, err)
	}
}
