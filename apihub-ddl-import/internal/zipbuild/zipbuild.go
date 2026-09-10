// Package zipbuild assembles the publish payload: the sources zip and the
// APIHUB BuildConfig JSON (see backend view/Build.go BuildConfig).
package zipbuild

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"sort"
)

// BuildConfig mirrors the fields of the backend's view.BuildConfig the tool sets.
type BuildConfig struct {
	PackageId                string               `json:"packageId"`
	Version                  string               `json:"version"`
	Status                   string               `json:"status"`
	PreviousVersion          string               `json:"previousVersion"`
	PreviousVersionPackageId string               `json:"previousVersionPackageId"`
	BuildType                string               `json:"buildType"`
	Files                    []BCFile             `json:"files"`
	Metadata                 *BuildConfigMetadata `json:"metadata,omitempty"`
}

type BCFile struct {
	FileId  string `json:"fileId"`
	Publish bool   `json:"publish"`
}

type BuildConfigMetadata struct {
	VersionLabels []string `json:"versionLabels,omitempty"`
}

// SourcesZip zips the enriched files (relpath → content) deterministically.
func SourcesZip(files map[string]string) ([]byte, error) {
	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for _, p := range paths {
		f, err := w.Create(p)
		if err != nil {
			return nil, err
		}
		if _, err := f.Write([]byte(files[p])); err != nil {
			return nil, err
		}
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ConfigJSON builds the publish config. previousVersion "none" (or "") publishes
// without a baseline; every file is published.
func ConfigJSON(packageID, version, status, previousVersion string, fileIDs, versionLabels []string) ([]byte, error) {
	if previousVersion == "none" {
		previousVersion = ""
	}
	cfg := BuildConfig{
		PackageId:                packageID,
		Version:                  version,
		Status:                   status,
		PreviousVersion:          previousVersion,
		PreviousVersionPackageId: "",
		BuildType:                "build",
		Files:                    make([]BCFile, 0, len(fileIDs)),
	}
	for _, id := range fileIDs {
		cfg.Files = append(cfg.Files, BCFile{FileId: id, Publish: true})
	}
	if len(versionLabels) > 0 {
		cfg.Metadata = &BuildConfigMetadata{VersionLabels: versionLabels}
	}
	return json.Marshal(cfg)
}
