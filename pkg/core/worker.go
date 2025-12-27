package core

type WorkerPool struct {
	tasks chan func()
}

func NewWorkerPool(size int) *WorkerPool {
	// Fail-fast: size must be positive
	if size <= 0 {
		FailFast(&EventBusError{Code: "INVALID_SIZE", Message: "worker pool size must be positive"})
	}
	wp := &WorkerPool{tasks: make(chan func(), 1000)}
	for i := 0; i < size; i++ {
		go func() {
			for task := range wp.tasks {
				task()
			}
		}()
	}
	return wp
}

func (wp *WorkerPool) Submit(task func()) {
	// Fail-fast: task cannot be nil
	if task == nil {
		FailFast(&EventBusError{Code: "INVALID_TASK", Message: "task cannot be nil"})
	}
	wp.tasks <- task
}

func (wp *WorkerPool) Shutdown() {
	close(wp.tasks)
}
