package tasks

import "sync"

type Task struct {
	PackageId string
	Version   string
}

func RunWorkers(workerCount int, tasks []Task, taskProcess func(Task)) {
	if len(tasks) == 0 {
		return
	}
	if workerCount < 1 {
		workerCount = 1
	}
	if workerCount > len(tasks) {
		workerCount = len(tasks)
	}

	wg := new(sync.WaitGroup)
	tasksCh := make(chan Task)

	for i := 0; i < workerCount; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			for task := range tasksCh {
				taskProcess(task)
			}
		}()
	}

	for _, task := range tasks {
		tasksCh <- task
	}

	close(tasksCh)
	wg.Wait()
}
