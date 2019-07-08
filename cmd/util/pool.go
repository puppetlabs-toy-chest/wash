package cmdutil

import (
	"sync"

	"github.com/gammazero/workerpool"
)

// Pool is a worker pool to limit work-in-progress and wait for a dynamic batch of work to
// complete. It uses WorkerPool to limit how many tasks run at once, and a wait group to know when
// all queued work is complete. Because the queue is dynamic, we need a wait group to tell us when
// all potential work is complete before we tell the pool to stop accepting new work and shutdown.
type Pool struct {
	wp *workerpool.WorkerPool
	wg *sync.WaitGroup
}

// NewPool creates a new dynamic worker pool.
func NewPool(parallel int) Pool {
	return Pool{wp: workerpool.New(parallel), wg: &sync.WaitGroup{}}
}

// Submit adds a new task to the pool.
func (p Pool) Submit(f func()) {
	p.wg.Add(1)
	p.wp.Submit(f)
}

// Done is used to mark that a task has completed.
func (p Pool) Done() {
	p.wg.Done()
}

// Finish waits until no more tasks are running, then shuts down the worker pool. This allows new
// tasks to be added to the pool after it's called by using a WaitGroup to determine when all tasks
// are done.
func (p Pool) Finish() {
	// Wait for the workgroup's we've queued to finish, then stop the worker pool.
	p.wg.Wait()
	p.wp.StopWait()
}
