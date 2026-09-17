package manifest

import (
	"apihub-bulk-src-fix/internal/tasks"
	"encoding/json"
	"os"
)

func Save(path string, tasksList []tasks.Task) error {
	data, err := json.MarshalIndent(tasksList, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func Read(path string) ([]tasks.Task, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var tasksList []tasks.Task

	if err := json.Unmarshal(data, &tasksList); err != nil {
		return nil, err
	}

	return tasksList, nil
}
