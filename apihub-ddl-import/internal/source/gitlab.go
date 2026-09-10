package source

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"apihub-ddl-import/internal/config"
	"apihub-ddl-import/internal/gitlab"
	"apihub-ddl-import/internal/model"
)

// InsecureTLS disables TLS certificate verification for GitLab fetches
// (wired from --insecure-skip-tls-verify).
var InsecureTLS bool

func gitlabDDL(src config.Source) ([]model.File, error) {
	c, err := gitlab.New(src.Repo, src.Token, InsecureTLS)
	if err != nil {
		return nil, err
	}
	dir := strings.Trim(src.Path, "/")
	entries, err := c.ListTree(dir, src.Branch)
	if err != nil {
		return nil, err
	}
	var files []model.File
	for _, e := range entries {
		if e.Type != "blob" || !strings.EqualFold(path.Ext(e.Path), ".sql") {
			continue
		}
		data, err := c.GetRawFile(e.Path, src.Branch)
		if err != nil {
			return nil, err
		}
		rel := strings.TrimPrefix(strings.TrimPrefix(e.Path, dir), "/")
		if rel == "" {
			rel = path.Base(e.Path)
		}
		files = append(files, model.File{RelPath: rel, Data: data})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].RelPath < files[j].RelPath })
	return files, nil
}

func gitlabComments(src config.Source) ([]byte, string, error) {
	c, err := gitlab.New(src.Repo, src.Token, InsecureTLS)
	if err != nil {
		return nil, "", err
	}
	filePath := strings.Trim(src.Path, "/")
	if filePath == "" {
		return nil, "", fmt.Errorf("commentsSource.path is empty")
	}
	data, err := c.GetRawFile(filePath, src.Branch)
	if err != nil {
		return nil, "", err
	}
	return data, path.Base(filePath), nil
}
