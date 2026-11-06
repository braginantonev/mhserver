package repository

type Semaphore interface {
	Acquire()
	Release()
}

type simpleSem struct {
	v chan struct{}
}

// Return simple semaphore
func NewSemaphore(size int) Semaphore {
	if size <= 0 {
		size = 1
	}
	return &simpleSem{v: make(chan struct{}, size)}
}

func (sem *simpleSem) Acquire() {
	sem.v <- struct{}{}
}

func (sem *simpleSem) Release() {
	<-sem.v
}
