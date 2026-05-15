package copy_test

import (
	"encoding/json"
	"testing"

	"apihub-portal-package-copy/internal/copy"
)

func TestTopologicalPublishOrder_linear_chain(t *testing.T) {
	items := []copy.PkgVer{
		{Pkg: "ws.a", Ver: "1"},
		{Pkg: "ws.a", Ver: "2"},
	}
	cfg1 := []byte(`{"previousVersion":"","previousVersionPackageId":""}`)
	cfg2 := []byte(`{"previousVersion":"1","previousVersionPackageId":""}`)
	configs := map[string][]byte{
		copy.ItemCfgKey("ws.a", "1"): cfg1,
		copy.ItemCfgKey("ws.a", "2"): cfg2,
	}
	order, warns, err := copy.TopologicalPublishOrder(items, configs)
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) != 0 {
		t.Fatalf("warnings: %v", warns)
	}
	if len(order) != 2 || order[0].Ver != "1" || order[1].Ver != "2" {
		t.Fatalf("order: %+v", order)
	}
}

func TestTopologicalPublishOrder_withRevisionIds(t *testing.T) {
	items := []copy.PkgVer{
		{Pkg: "ws.a", Ver: "2025.2@1"},
		{Pkg: "ws.a", Ver: "2025.3@2"},
	}
	cfg1 := []byte(`{"previousVersion":"","previousVersionPackageId":""}`)
	cfg2 := []byte(`{"previousVersion":"2025.2@1","previousVersionPackageId":""}`)
	configs := map[string][]byte{
		copy.ItemCfgKey("ws.a", "2025.2@1"): cfg1,
		copy.ItemCfgKey("ws.a", "2025.3@2"): cfg2,
	}
	order, warns, err := copy.TopologicalPublishOrder(items, configs)
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) != 0 {
		t.Fatalf("warnings: %v", warns)
	}
	if len(order) != 2 || order[0].Ver != "2025.2@1" || order[1].Ver != "2025.3@2" {
		t.Fatalf("order: %+v", order)
	}
}

func TestTopologicalPublishOrder_logicalPrevMatchesRevisionRow(t *testing.T) {
	items := []copy.PkgVer{
		{Pkg: "ws.a", Ver: "2025.2@1"},
		{Pkg: "ws.a", Ver: "2025.3@2"},
	}
	cfg1 := []byte(`{"previousVersion":"","previousVersionPackageId":""}`)
	cfg2 := []byte(`{"previousVersion":"2025.2","previousVersionPackageId":""}`)
	configs := map[string][]byte{
		copy.ItemCfgKey("ws.a", "2025.2@1"): cfg1,
		copy.ItemCfgKey("ws.a", "2025.3@2"): cfg2,
	}
	order, warns, err := copy.TopologicalPublishOrder(items, configs)
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) != 0 {
		t.Fatalf("warnings: %v", warns)
	}
	if len(order) != 2 || order[0].Ver != "2025.2@1" || order[1].Ver != "2025.3@2" {
		t.Fatalf("order: %+v", order)
	}
}

func TestTopologicalPublishOrder_missing_prev_warns(t *testing.T) {
	items := []copy.PkgVer{{Pkg: "ws.a", Ver: "2"}}
	cfg2 := []byte(`{"previousVersion":"1","previousVersionPackageId":""}`)
	configs := map[string][]byte{copy.ItemCfgKey("ws.a", "2"): cfg2}
	order, warns, err := copy.TopologicalPublishOrder(items, configs)
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) != 1 || len(order) != 1 {
		t.Fatalf("warns=%v order=%+v", warns, order)
	}
}

func TestRemapBuildConfig_stripsRevisionFields(t *testing.T) {
	raw := []byte(`{"packageId":"p","version":"2025.4@2","previousVersion":"2025.3@9","previousVersionPackageId":"","refs":[]}`)
	prevKept := []copy.PkgVer{{Pkg: "p", Ver: "2025.4@2"}, {Pkg: "p", Ver: "2025.3@9"}}
	out, err := copy.RemapBuildConfig(raw, "p", "tgt", map[string]string{}, prevKept)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if m["version"] != "2025.4" {
		t.Fatalf("version got %v", m["version"])
	}
	if m["previousVersion"] != "2025.3" {
		t.Fatalf("previousVersion got %v", m["previousVersion"])
	}
	if m["packageId"] != "tgt" {
		t.Fatal()
	}
}

func TestRemapBuildConfig_dropsPreviousNotInCopiedSet(t *testing.T) {
	raw := []byte(`{"packageId":"p","version":"2025.2@1","previousVersion":"2025.1","previousVersionPackageId":"","refs":[]}`)
	copied := []copy.PkgVer{{Pkg: "p", Ver: "2025.2@1"}}
	out, err := copy.RemapBuildConfig(raw, "p", "tgt", map[string]string{}, copied)
	if err != nil {
		t.Fatal(err)
	}
	var mc map[string]any
	if err := json.Unmarshal(out, &mc); err != nil {
		t.Fatal(err)
	}
	if mc["previousVersion"] != "" {
		t.Fatalf("expected empty previousVersion, got %v", mc["previousVersion"])
	}
	if mc["version"] != "2025.2" {
		t.Fatalf("version got %v", mc["version"])
	}
}
