package file

import (
	"apihub-bulk-src-fix/internal/tasks"
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func ReadTasksFromFile(path string) ([]tasks.Task, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var tasksList []tasks.Task

	scanner := bufio.NewScanner(file)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++

		parts := strings.Split(scanner.Text(), " ")

		if len(parts) != 2 {
			continue
		}

		if !strings.Contains(parts[1], "@") {
			return nil, fmt.Errorf(
				"%s:%d: version %q has no revision, use <version>@<revision>",
				path, lineNumber, parts[1],
			)
		}

		tasksList = append(tasksList, tasks.Task{
			PackageId: parts[0],
			Version:   parts[1],
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return tasksList, nil
}

func SaveZipIntoFolder(folder, filename string, data []byte) error {
	if err := os.MkdirAll(folder, 0755); err != nil {
		return err
	}

	path := filepath.Join(folder, filename)

	return os.WriteFile(path, data, 0644)
}

func UploadZipFile(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}

func Unzip(source, destination string) error {
	archive, err := zip.OpenReader(source)
	if err != nil {
		return err
	}
	defer archive.Close()

	for _, file := range archive.File {
		path := filepath.Join(destination, file.Name)

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(path, 0755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}

		src, err := file.Open()
		if err != nil {
			return err
		}

		dst, err := os.Create(path)
		if err != nil {
			src.Close()
			return err
		}

		_, err = io.Copy(dst, src)

		src.Close()
		dst.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

func Zip(source, destination string) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return err
	}

	dst, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer dst.Close()

	archive := zip.NewWriter(dst)
	defer archive.Close()

	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relativePath, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}

		writer, err := archive.Create(relativePath)
		if err != nil {
			return err
		}

		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()

		_, err = io.Copy(writer, src)

		return err
	})
}
