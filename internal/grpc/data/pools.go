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

type DataPool[T any, R any] struct {
	tasks   chan T
	errs    chan error
	results chan R
	file    File

	wg *sync.WaitGroup
}

// results_needed param is temp. I hope I change this
func NewDataPool[T any, R any](sem Semaphore, file File, results_needed bool, work_func func(T) (R, error)) *DataPool[T, R] {
	tasks := make(chan T, MAX_TASKS)
	results := make(chan R, MAX_TASKS)
	errs := make(chan error, MAX_TASKS)

	wg := sync.WaitGroup{}
	wg.Add(WORKERS_COUNT)
	for range WORKERS_COUNT {
		go func() {
			defer wg.Done()
			for task := range tasks {
				sem <- struct{}{}
				res, err := work_func(task)
				if err != nil {
					errs <- err
					continue
				}

				if results_needed {
					results <- res
				}

				<-sem
			}
		}()
	}

	return &DataPool[T, R]{
		tasks:   tasks,
		errs:    errs,
		results: results,
		file:    file,
		wg:      &wg,
	}
}

func (self *DataPool[T, R]) Push(task T) {
	self.tasks <- task
}

func (self *DataPool[T, R]) Results() <-chan R {
	return self.results
}

func (self *DataPool[T, R]) Errors() <-chan error {
	return self.errs
}

func (self *DataPool[T, R]) Flush() {
	close(self.tasks)
	self.wg.Wait()

	close(self.errs)
	close(self.results)
}

type SavePool struct {
	*DataPool[*pb.Chunk, struct{}] // empty result
}

func NewSavePool(ctx context.Context, sem Semaphore, file File) SavePool {
	return SavePool{
		DataPool: NewDataPool(sem, file, false, func(c *pb.Chunk) (struct{}, error) {
			if uint64(len(c.Data))+c.Offset > file.Meta.Size {
				return struct{}{}, ErrUnexpectedFileChange
			}

			slog.Info("save", slog.String("data", string(c.Data)))

			_, err := file.WriteAt(c.Data, int64(c.Offset))
			if err != nil {
				slog.Error("failed save chunk", slog.Any("error", err))
			}
			return struct{}{}, err
		}),
	}
}

type ReadPool = *DataPool[uint64, *pb.Chunk]
