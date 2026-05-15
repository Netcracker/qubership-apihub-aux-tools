package scope_test

import (
	"testing"

	"apihub-portal-package-copy/internal/scope"
)

func TestNormalizeVersionTokens_starOnly(t *testing.T) {
	v, err := scope.NormalizeVersionTokens([]string{"  * ", ""})
	if err != nil {
		t.Fatal(err)
	}
	if len(v) != 1 || v[0] != "*" {
		t.Fatalf("%v", v)
	}
	_, err = scope.NormalizeVersionTokens([]string{"*", "1.0"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestQualifyUnderWorkspace(t *testing.T) {
	cases := []struct {
		ws, id, want string
	}{
		{"ws.a", "*", "*"},
		{"ws.a", "ws.a", "ws.a"},
		{"ws.a", "ws.a.pkg", "ws.a.pkg"},
		{"ws.a.b", "pkg", "ws.a.b.pkg"},
		{"ws", "grp.sub", "ws.grp.sub"},
		{"ws", "", ""},
		{"", "x", "x"},
	}
	for _, tc := range cases {
		got := scope.QualifyUnderWorkspace(tc.ws, tc.id)
		if got != tc.want {
			t.Fatalf("QualifyUnderWorkspace(%q,%q) = %q, want %q", tc.ws, tc.id, got, tc.want)
		}
	}
	ex := scope.QualifyExcludedUnderWorkspace("ws.a", []string{"grp", "ws.a.keep", ""})
	want := []string{"ws.a.grp", "ws.a.keep"}
	if len(ex) != len(want) {
		t.Fatalf("exclusions: %v", ex)
	}
	for i := range want {
		if ex[i] != want[i] {
			t.Fatalf("exclusions[%d] got %q want %q", i, ex[i], want[i])
		}
	}
}

func TestPackageExcluded(t *testing.T) {
	ex := []string{"ws.grp", "ws.other.pkg"}
	if !scope.PackageExcluded("ws.grp.pkg1", ex) {
		t.Fatal()
	}
	if !scope.PackageExcluded("ws.grp", ex) {
		t.Fatal()
	}
	if scope.PackageExcluded("ws.grp_other", ex) {
		t.Fatal()
	}
}
