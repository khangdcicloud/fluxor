package core

type WorkerPool struct {
	tasks chan func()
}

func NewWorkerPool(size int) *WorkerPool {
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
	wp.tasks <- task
}

func (wp *WorkerPool) Shutdown() {
	close(wp.tasks)
}
