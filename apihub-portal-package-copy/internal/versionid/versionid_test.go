package versionid_test

import (
	"testing"

	"apihub-portal-package-copy/internal/versionid"
)

func TestSpecMatchesPublished(t *testing.T) {
	cases := []struct {
		spec, api string
		want      bool
	}{
		{"2025.4", "2025.4@1", true},
		{"2025.4", "2025.4", true},
		{"2025.4@1", "2025.4@1", true},
		{"2025.4@1", "2025.4@2", false},
		{"2025.4", "2025.40@1", false},
		{"2025.4", "2025.10@1", false},
		{" 2025.4 ", "2025.4@1", true},
	}
	for _, tc := range cases {
		got := versionid.SpecMatchesPublished(tc.spec, tc.api)
		if got != tc.want {
			t.Fatalf("SpecMatchesPublished(%q,%q)=%v want %v", tc.spec, tc.api, got, tc.want)
		}
	}
}

func TestBaseOnly(t *testing.T) {
	if versionid.BaseOnly("2025.4@3") != "2025.4" {
		t.Fatal()
	}
	if versionid.BaseOnly("2025.4") != "2025.4" {
		t.Fatal()
	}
}

func TestPickPublished_multi(t *testing.T) {
	r, ok := versionid.PickPublished("2025.4", []string{"2025.4@1", "2025.4@3", "2025.3@9"})
	if !ok || r != "2025.4@3" {
		t.Fatalf("got %q ok=%v", r, ok)
	}
}
