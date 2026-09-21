package groups

import (
	"net/http"
	"net/http/httptest"
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		r.Body.Read(buf)
		gotBody = string(buf)
		w.WriteHeader(http.StatusCreated)
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
}
