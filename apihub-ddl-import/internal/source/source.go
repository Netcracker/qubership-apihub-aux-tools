// Package source fetches the two inputs (DDL .sql files and the comments
// workbook) from a local path or a GitLab repository.
package source

import (
	"fmt"

	"apihub-ddl-import/internal/config"
	"apihub-ddl-import/internal/model"
)

// FetchDDL returns all .sql files of the DDL source. RelPath uses forward
// slashes and is later reused as the APIHUB fileId inside the sources zip.
func FetchDDL(src config.Source) ([]model.File, error) {
	switch src.Type {
	case config.SourceFile:
		return localDDL(src.Path)
	case config.SourceGitlab:
		return gitlabDDL(src)
	}
	return nil, fmt.Errorf("unknown ddlSource type %q", src.Type)
}

// FetchComments returns the workbook bytes and a display name for artifacts.
func FetchComments(src config.Source) ([]byte, string, error) {
	switch src.Type {
	case config.SourceFile:
		return localComments(src.Path)
	case config.SourceGitlab:
		return gitlabComments(src)
	}
	return nil, "", fmt.Errorf("unknown commentsSource type %q", src.Type)
}
