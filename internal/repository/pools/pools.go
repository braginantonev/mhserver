package pools

import (
	"context"
	"sync"
)

const (
	WORKERS_COUNT int = 5
	MAX_TASKS     int = WORKERS_COUNT * 2
)

// The standard workers pool for implementation.
// In first idea, I use this pool to create data workers pools: SaveWorkersPool and ReadWorkersPool
type WorkersPool[T any] struct {
	tasks chan T

	closeOnce sync.Once
	cancel    context.CancelCauseFunc
	ctx       context.Context

	wg sync.WaitGroup
}

func NewWorkersPool[T any](ctx context.Context, handle_task func(T) error) *WorkersPool[T] {
	tasks := make(chan T, MAX_TASKS)

	pool_ctx, cancel := context.WithCancelCause(ctx)
	pool := WorkersPool[T]{
		tasks:  tasks,
		cancel: cancel,
		ctx:    pool_ctx,
		wg:     sync.WaitGroup{},
	}

	for range WORKERS_COUNT {
		pool.wg.Go(func() {
			for {
				select {
				case <-pool_ctx.Done():
					return
				case task, ok := <-tasks:
					if !ok {
						return
					}

					if err := handle_task(task); err != nil {
						pool.handleError(err)
					}
				}
			}
		})
	}

	return &pool
}

func (self *WorkersPool[T]) handleError(err error) {
	self.closeOnce.Do(func() {
		close(self.tasks)
	})
	self.cancel(err)
}

func (self *WorkersPool[T]) TryPush(task T) error {
	select {
	case <-self.ctx.Done():
		return self.Recover()
	case self.tasks <- task:
		return nil
	}
}

func (self *WorkersPool[T]) Flush() {
	self.closeOnce.Do(func() {
		close(self.tasks)
	})
	self.wg.Wait()
}

func (self *WorkersPool[T]) Recover() error {
	if err := context.Cause(self.ctx); err != context.Canceled {
		return err
	}
	return nil
}
