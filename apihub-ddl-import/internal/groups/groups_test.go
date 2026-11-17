package groups

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"apihub-ddl-import/internal/apihub"
	"apihub-ddl-import/internal/model"
)

func newClient(t *testing.T, srv *httptest.Server) *apihub.Client {
	t.Helper()
	c := apihub.New(srv.URL, "test-key", false)
	c.HTTPClient = srv.Client()
	return c
}

func testResult() (*model.MergeResult, []apihub.DdlEntity) {
	res := &model.MergeResult{
		DomainByTable: map[string]string{model.NormKey("t1"): "ACM"},
		Domains:       []string{"ACM"},
	}
	entities := []apihub.DdlEntity{{DdlEntityId: "e1", Name: "t1"}}
	return res, entities
}

// TestCreateMisdirectedRequestTreatedAsUnavailable locks in a live-observed
// backend behavior: some deployed APIHUB backends answer an unmounted
// /ddl/groups route with 421 "Requested unknown endpoint" rather than a
// plain 404/405. This must degrade the groups step exactly like a 404 would
// (a single GROUPS_API_UNAVAILABLE warning), not report a per-domain failure.
func TestCreateMisdirectedRequestTreatedAsUnavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusMisdirectedRequest)
		w.Write([]byte(`{"status":421,"message":"Requested unknown endpoint"}`))
	}))
	defer srv.Close()

	res, entities := testResult()
	out := Create(newClient(t, srv), "pkg", "v1", res, entities, "Tables of the %s domain")

	if out.APIAvailable {
		t.Error("APIAvailable must be false after a 421 response")
	}
	if len(out.Created) != 0 || len(out.Failed) != 0 {
		t.Errorf("no per-domain outcome expected, got created=%v failed=%v", out.Created, out.Failed)
	}
	if len(out.Warnings) != 1 || out.Warnings[0].Code != model.WGroupsApiUnavailable {
		t.Errorf("warnings = %+v, want exactly one GROUPS_API_UNAVAILABLE", out.Warnings)
	}
}

func TestCreateSuccess(t *testing.T) {
	var gotBody string
	var sawPatch bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		r.Body.Read(buf)
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"groups":[]}`))
		case http.MethodPost:
			gotBody = string(buf)
			w.WriteHeader(http.StatusCreated)
		case http.MethodPatch:
			sawPatch = true
			if !bytes.Contains(buf, []byte(`"tables":[{"ddlEntityId":"e1"}]`)) {
				t.Errorf("patch body = %s", buf)
			}
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	}))
	defer srv.Close()

	res, entities := testResult()
	out := Create(newClient(t, srv), "pkg", "v1", res, entities, "Tables of the %s domain")

	if !out.APIAvailable {
		t.Error("APIAvailable must stay true on success")
	}
	if len(out.Created) != 1 || out.Created[0] != "ACM" {
		t.Errorf("created = %v, want [ACM]", out.Created)
	}
	if len(out.Warnings) != 0 {
		t.Errorf("warnings = %v, want none", out.Warnings)
	}
	if gotBody == "" {
		t.Error("expected a request body")
	}
	// The create endpoint ignores tables; membership must be set with a
	// follow-up update, or a freshly created group is left empty.
	if !sawPatch {
		t.Error("expected a follow-up PATCH to set the group's tables")
	}
}

func TestCreateNoGroupsCopied(t *testing.T) {
	posts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"groups":[]}`))
		case http.MethodPost:
			posts++
			w.WriteHeader(http.StatusCreated)
		case http.MethodPatch:
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	}))
	defer srv.Close()

	res, entities := twoDomains()
	out := Create(newClient(t, srv), "pkg", "v1", res, entities, "Tables of the %s domain")
	if posts != 2 || len(out.Created) != 2 || len(out.Updated) != 0 || len(out.Failed) != 0 {
		t.Errorf("posts=%d created=%v updated=%v failed=%v", posts, out.Created, out.Updated, out.Failed)
	}
}

func TestCreatePartialGroupsCopied(t *testing.T) {
	var posted []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			if strings.Contains(r.URL.Path, "/ddl/groups/") {
				w.Write([]byte(`{"entities":[{"ddlEntityId":"e1"}]}`))
				return
			}
			w.Write([]byte(`{"groups":[{"groupName":"ACM","tablesCount":1}]}`))
		case http.MethodPost:
			var req apihub.CreateDdlGroupReq
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("decode create body: %v", err)
			}
			posted = append(posted, req.GroupName)
			w.WriteHeader(http.StatusCreated)
		case http.MethodPatch:
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	}))
	defer srv.Close()

	res, entities := twoDomains()
	out := Create(newClient(t, srv), "pkg", "v1", res, entities, "Tables of the %s domain")
	if len(posted) != 1 || posted[0] != "BILL" {
		t.Errorf("posted = %v, want [BILL]", posted)
	}
	if len(out.Created) != 1 || out.Created[0] != "BILL" {
		t.Errorf("created = %v, want [BILL]", out.Created)
	}
	if len(out.Unchanged) != 1 || out.Unchanged[0] != "ACM" {
		t.Errorf("unchanged = %v, want [ACM]", out.Unchanged)
	}
	if len(out.Updated) != 0 {
		t.Errorf("updated = %v, want none", out.Updated)
	}
	if len(out.Failed) != 0 {
		t.Errorf("failed = %v", out.Failed)
	}
}

func TestCreateAllGroupsCopied(t *testing.T) {
	posts := 0
	patches := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			switch {
			case strings.HasSuffix(r.URL.Path, "/ACM"):
				w.Write([]byte(`{"entities":[{"ddlEntityId":"e1"}]}`))
			case strings.HasSuffix(r.URL.Path, "/BILL"):
				w.Write([]byte(`{"entities":[{"ddlEntityId":"e2"}]}`))
			default:
				w.Write([]byte(`{"groups":[{"groupName":"ACM"},{"groupName":"BILL"}]}`))
			}
		case http.MethodPost:
			posts++
			w.WriteHeader(http.StatusCreated)
		case http.MethodPatch:
			patches++
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	}))
	defer srv.Close()

	res, entities := twoDomains()
	out := Create(newClient(t, srv), "pkg", "v1", res, entities, "Tables of the %s domain")
	if posts != 0 || patches != 0 {
		t.Errorf("posts=%d patches=%d, want no writes", posts, patches)
	}
	if len(out.Unchanged) != 2 || len(out.Updated) != 0 || len(out.Created) != 0 || len(out.Failed) != 0 {
		t.Errorf("created=%v updated=%v unchanged=%v failed=%v", out.Created, out.Updated, out.Unchanged, out.Failed)
	}
}

func TestCreateCopiedGroupContentReplaced(t *testing.T) {
	patched := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			if strings.Contains(r.URL.Path, "/ddl/groups/") {
				if patched {
					w.Write([]byte(`{"entities":[{"ddlEntityId":"e1"}]}`))
					return
				}
				w.Write([]byte(`{"entities":[{"ddlEntityId":"e-old"},{"ddlEntityId":"e1"}]}`))
				return
			}
			w.Write([]byte(`{"groups":[{"groupName":"ACM","tablesCount":2}]}`))
		case http.MethodPatch:
			patched = true
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	}))
	defer srv.Close()

	res, entities := testResult()
	out := Create(newClient(t, srv), "pkg", "v1", res, entities, "Tables of the %s domain")
	if !patched || len(out.Updated) != 1 || len(out.Unchanged) != 0 || len(out.Failed) != 0 {
		t.Errorf("patched=%v updated=%v unchanged=%v failed=%v", patched, out.Updated, out.Unchanged, out.Failed)
	}
}

func TestCreateCopiedGroupContentStillWrong(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			if strings.Contains(r.URL.Path, "/ddl/groups/") {
				w.Write([]byte(`{"entities":[{"ddlEntityId":"e-old"}]}`))
				return
			}
			w.Write([]byte(`{"groups":[{"groupName":"ACM"}]}`))
		case http.MethodPatch:
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	}))
	defer srv.Close()

	res, entities := testResult()
	out := Create(newClient(t, srv), "pkg", "v1", res, entities, "Tables of the %s domain")
	if len(out.Failed) != 1 || out.Failed[0] != "ACM" || len(out.Updated) != 0 {
		t.Errorf("updated=%v failed=%v warnings=%v", out.Updated, out.Failed, out.Warnings)
	}
}

func TestCreateDuplicateAfterEmptyListIsUpdate(t *testing.T) {
	patched := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			if strings.Contains(r.URL.Path, "/ddl/groups/") {
				body := `{"entities":[{"ddlEntityId":"e-old"}]}`
				if patched {
					body = `{"entities":[{"ddlEntityId":"e1"}]}`
				}
				w.Write([]byte(body))
				return
			}
			w.Write([]byte(`{"groups":[]}`))
		case http.MethodPost:
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"status":400,"code":"8403","message":"DDL table group with groupName=ACM already exists"}`))
		case http.MethodPatch:
			patched = true
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	}))
	defer srv.Close()

	res, entities := testResult()
	out := Create(newClient(t, srv), "pkg", "v1", res, entities, "Tables of the %s domain")
	if len(out.Updated) != 1 || out.Updated[0] != "ACM" || len(out.Created) != 0 || len(out.Failed) != 0 {
		t.Errorf("created=%v updated=%v failed=%v", out.Created, out.Updated, out.Failed)
	}
}

func twoDomains() (*model.MergeResult, []apihub.DdlEntity) {
	res := &model.MergeResult{
		DomainByTable: map[string]string{
			model.NormKey("t1"): "ACM",
			model.NormKey("t2"): "BILL",
		},
		Domains: []string{"ACM", "BILL"},
	}
	entities := []apihub.DdlEntity{
		{DdlEntityId: "e1", Name: "t1"},
		{DdlEntityId: "e2", Name: "t2"},
	}
	return res, entities
}
