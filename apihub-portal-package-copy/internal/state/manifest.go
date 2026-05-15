package state

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const ManifestFile = "manifest.json"

type Manifest struct {
	ManifestVersion   int               `json:"manifestVersion"`
	FetchComplete     bool              `json:"fetchComplete"`
	SourceWorkspaceID string            `json:"sourceWorkspaceId"`
	TargetWorkspaceID string            `json:"targetWorkspaceId"`
	SourceRootID      string            `json:"sourceRootId"`
	DesiredVersions   []string          `json:"desiredVersions"`
	ExcludedPackages  []string          `json:"excludedPackages,omitempty"`
	PackageMap        map[string]string `json:"packageMap"` // source package id -> target package id
	Items             []WorkItem        `json:"items"`
}

type WorkItem struct {
	SourcePackageID string `json:"sourcePackageId"`
	Version         string `json:"version"`
	Subdir          string `json:"subdir"` // under data/

	FetchDone  bool   `json:"fetchDone"`
	FetchError string `json:"fetchError,omitempty"`
	Published  bool   `json:"published"`
	PublishErr string `json:"publishError,omitempty"`
	PublishID  string `json:"publishId,omitempty"`
}

func LoadManifest(dir string) (*Manifest, error) {
	p := filepath.Join(dir, ManifestFile)
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	if m.PackageMap == nil {
		m.PackageMap = map[string]string{}
	}
	if m.ExcludedPackages == nil {
		m.ExcludedPackages = []string{}
	}
	return &m, nil
}

func SaveManifest(dir string, m *Manifest) error {
	if m.PackageMap == nil {
		m.PackageMap = map[string]string{}
	}
	if m.ExcludedPackages == nil {
		m.ExcludedPackages = []string{}
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	p := filepath.Join(dir, ManifestFile)
	return os.WriteFile(p, b, 0644)
}

func NewManifest(wsSrc, wsTgt, root string, versions []string, excludedPackages []string) *Manifest {
	return &Manifest{
		ManifestVersion:   4,
		SourceWorkspaceID: wsSrc,
		TargetWorkspaceID: wsTgt,
		SourceRootID:      root,
		DesiredVersions:   append([]string(nil), versions...),
		ExcludedPackages:  append([]string(nil), excludedPackages...),
		PackageMap:        map[string]string{},
		Items:             nil,
	}
}
