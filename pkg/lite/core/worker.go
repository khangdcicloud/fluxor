package core

import "context"

type WorkerPool struct {
	tasks chan func(context.Context)
}

func NewWorkerPool(workerCount int, queueSize int) *WorkerPool {
	if workerCount <= 0 {
		workerCount = 1
	}
	if queueSize <= 0 {
		queueSize = 1024
	}

	wp := &WorkerPool{tasks: make(chan func(context.Context), queueSize)}
	for i := 0; i < workerCount; i++ {
		go func() {
			for task := range wp.tasks {
				task(context.Background())
			}
		}()
	}
	return wp
}

func (wp *WorkerPool) Submit(task func(context.Context)) {
	wp.tasks <- task
}

func (wp *WorkerPool) Shutdown() {
	close(wp.tasks)
}
