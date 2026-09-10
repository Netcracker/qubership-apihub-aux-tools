package zipbuild

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"testing"
)

func TestSourcesZip(t *testing.T) {
	data, err := SourcesZip(map[string]string{
		"b/second.sql": "CREATE TABLE b();",
		"a/first.sql":  "CREATE TABLE a();",
	})
	if err != nil {
		t.Fatal(err)
	}
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	if len(r.File) != 2 || r.File[0].Name != "a/first.sql" || r.File[1].Name != "b/second.sql" {
		t.Fatalf("zip entries: %v", []string{r.File[0].Name, r.File[1].Name})
	}
	rc, _ := r.File[0].Open()
	content, _ := io.ReadAll(rc)
	rc.Close()
	if string(content) != "CREATE TABLE a();" {
		t.Errorf("content = %q", content)
	}
}

func TestConfigJSON(t *testing.T) {
	data, err := ConfigJSON("pkg.a", "2026.2", "draft", "2026.1", []string{"db/base.sql"}, []string{"L1"})
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg["packageId"] != "pkg.a" || cfg["version"] != "2026.2" || cfg["status"] != "draft" ||
		cfg["previousVersion"] != "2026.1" || cfg["buildType"] != "build" {
		t.Errorf("cfg = %v", cfg)
	}
	files := cfg["files"].([]any)
	f0 := files[0].(map[string]any)
	if f0["fileId"] != "db/base.sql" || f0["publish"] != true {
		t.Errorf("files[0] = %v", f0)
	}
	meta := cfg["metadata"].(map[string]any)
	labels := meta["versionLabels"].([]any)
	if len(labels) != 1 || labels[0] != "L1" {
		t.Errorf("labels = %v", labels)
	}
}

func TestConfigJSONNonePreviousVersion(t *testing.T) {
	data, err := ConfigJSON("pkg.a", "2026.1", "draft", "none", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	json.Unmarshal(data, &cfg)
	if cfg["previousVersion"] != "" {
		t.Errorf("previousVersion = %q, want empty for 'none'", cfg["previousVersion"])
	}
	if _, ok := cfg["metadata"]; ok {
		t.Error("metadata must be omitted when no labels")
	}
}
