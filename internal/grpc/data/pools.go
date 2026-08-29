package data

import (
	"context"
	"io"
	"log/slog"
	"sync"

	"github.com/braginantonev/mhserver/internal/repository"
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
						pool.handleError(err)
					}
				}
			}
		})
	}

	return &pool
}

func (self *DataWorkersPool[T]) handleError(err error) {
	self.closeOnce.Do(func() {
		close(self.tasks)
	})
	self.cancel(err)
}

func (self *DataWorkersPool[T]) TryPush(task T) error {
	select {
	case <-self.ctx.Done():
		return self.Recover()
	case self.tasks <- task:
		return nil
	}
}

func (self *DataWorkersPool[T]) Flush() {
	self.closeOnce.Do(func() {
		close(self.tasks)
	})
	self.wg.Wait()
}

// Use this method if you pool will be used for writing in file
// For read or another actions, where file sync is not needed - just use `Flush()`
func (self *DataWorkersPool[T]) FlushAndSync() error {
	self.Flush()
	return self.file.Sync()
}

func (self *DataWorkersPool[T]) Recover() error {
	if err := context.Cause(self.ctx); err != context.Canceled {
		return err
	}
	return nil
}

type SaveWorkersPool struct {
	*DataWorkersPool[*pb.Chunk]
}

func NewSaveWorkersPool(ctx context.Context, service_sem repository.Semaphore, file File) *SaveWorkersPool {
	return &SaveWorkersPool{
		NewDataWorkersPool(ctx, file, func(c *pb.Chunk) error {
			defer service_sem.Release()
			service_sem.Acquire()

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

type ReadWorkersPool struct {
	*DataWorkersPool[uint64] // offsets
	results                  chan *pb.Chunk
}

func NewReadWorkersPool(
	ctx context.Context,
	service_sem repository.Semaphore,
	chunk_size uint64,
	file File,
) *ReadWorkersPool {
	results := make(chan *pb.Chunk, MAX_TASKS)
	return &ReadWorkersPool{
		DataWorkersPool: NewDataWorkersPool(ctx, file, func(offset uint64) error {
			defer service_sem.Release()
			service_sem.Acquire()

			data := make([]byte, offset)
			n, err := file.ReadAt(data, int64(offset))
			if err != nil && err != io.EOF {
				slog.ErrorContext(ctx, "failed read file chunk", slog.Any("err", err))
				return ErrInternal
			}

			if n == 0 && err == io.EOF {
				return ErrReadOutOfFile
			}

			results <- &pb.Chunk{
				Data:   data,
				Offset: offset,
			}

			if err == io.EOF {
				close(results)
			}
			return nil
		}),
		results: results,
	}
}
