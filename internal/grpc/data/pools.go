package data

import (
	"context"
	"log/slog"
	"sync"

	pb "github.com/braginantonev/mhserver/proto/data"
)

const (
	WORKERS_COUNT int = 5
	MAX_TASKS     int = WORKERS_COUNT * 2
)

type DataWorkersPool[T any] struct {
	tasks chan T
	file  File

	closeOnce sync.Once
	cancel    context.CancelCauseFunc
	ctx       context.Context

	wg sync.WaitGroup
}

func NewDataWorkersPool[T any](ctx context.Context, file File, handle_task func(T) error) *DataWorkersPool[T] {
	tasks := make(chan T, MAX_TASKS)

	pool_ctx, cancel := context.WithCancelCause(ctx)
	pool := DataWorkersPool[T]{
		tasks:  tasks,
		file:   file,
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
						pool.emitError(err)
					}
				}
			}

		})
	}

	return &pool
}

func (self *DataWorkersPool[T]) emitError(err error) {
	self.Close()
}

func (self *DataWorkersPool[T]) TryPush(task T) error {
	select {
	case <-self.ctx.Done():
		return context.Cause(self.ctx)
	case self.tasks <- task:
		return nil
	}
}

func (self *DataWorkersPool[T]) CloseCause(err error) {
	self.closeOnce.Do(func() {
		self.cancel(err)
		close(self.tasks)
	})
}

func (self *DataWorkersPool[T]) Close() {
	self.closeOnce.Do(func() {
		self.cancel(nil)
		close(self.tasks)
	})
}

func (self *DataWorkersPool[T]) Flush(with_sync bool) error {
	self.wg.Wait()
	if with_sync {
		return self.file.Sync()
	}
	return nil
}

type SaveWorkersPool struct {
	*DataWorkersPool[*pb.Chunk]
}

func NewSaveWorkersPool(ctx context.Context, service_sem Semaphore, file File) *SaveWorkersPool {
	return &SaveWorkersPool{
		NewDataWorkersPool(ctx, file, func(c *pb.Chunk) error {
			defer func() { <-service_sem }()
			service_sem <- struct{}{}

			if uint64(len(c.Data))+c.Offset > file.Meta.Size {
				return ErrUnexpectedFileChange
			}

			_, err := file.WriteAt(c.Data, int64(c.Offset))
			if err != nil {
				slog.ErrorContext(ctx, "failed save chunk", slog.Any("error", err))
				return ErrInternal
			}

			return nil
		}),
	}

}

// type ReadPool = *DataWorkersPool[uint64]
