package source

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"apihub-ddl-import/internal/model"
)

func localDDL(path string) ([]model.File, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("ddl source %s: %w", path, err)
	}
	if !info.IsDir() {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		return []model.File{{RelPath: filepath.Base(path), Data: data}}, nil
	}
	var files []model.File
	err = filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.EqualFold(filepath.Ext(p), ".sql") {
			return nil
		}
		rel, err := filepath.Rel(path, p)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		files = append(files, model.File{RelPath: filepath.ToSlash(rel), Data: data})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk ddl source %s: %w", path, err)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].RelPath < files[j].RelPath })
	return files, nil
}

func localComments(path string) ([]byte, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("comments source %s: %w", path, err)
	}
	return data, filepath.Base(path), nil
}
