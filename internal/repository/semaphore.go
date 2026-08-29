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

func (self *simpleSem) Acquire() {
	self.v <- struct{}{} // Второй вариант
}

func (self *simpleSem) Release() {
	<-self.v
}
