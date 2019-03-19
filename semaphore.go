package semaphore

import (
	"sync"
)

type Semaphore struct {
	sync.RWMutex
	sem          chan struct{}
	size         uint64
	newSize      uint64
	semOverflow  chan struct{}
	resizing     chan struct{}
	doneResizing chan struct{}
}

func NewSemaphore(size uint64) *Semaphore {
	return &Semaphore{
		sem:          make(chan struct{}, size),
		size:         size,
		semOverflow:  make(chan struct{}, 0),
		resizing:     make(chan struct{}),
		doneResizing: make(chan struct{}),
	}
}

func (s *Semaphore) resize() {
	s.Lock()

	defer s.Unlock()
	defer close(s.doneResizing)

	if s.newSize == s.size {
		return
	}

	s.resizing = make(chan struct{})
	semCounter := len(s.sem) + len(s.semOverflow)
	s.sem = make(chan struct{}, s.newSize)
	if uint64(semCounter) > s.newSize {
		s.semOverflow = make(chan struct{}, uint64(semCounter)-s.newSize)
	} else {
		s.semOverflow = make(chan struct{}, 0)
	}

	for i := semCounter; i > 0; i-- {
		select {
		case s.sem <- struct{}{}:
			break
		default:
			select {
			case s.semOverflow <- struct{}{}:
				break
			default:
				panic("semoverflow is overflowed!!!")
			}

		}
	}

	s.size = s.newSize

}

func (s *Semaphore) Resize(newSize uint64) {
	if newSize == s.size {
		return
	}
	s.newSize = newSize
	close(s.resizing)
	<-s.doneResizing
	s.doneResizing = make(chan struct{})
}

func (s *Semaphore) Acquire() {
	s.RLock()
	select {
	case s.sem <- struct{}{}:
		s.RUnlock()
		break
	case <-s.resizing:
		s.RUnlock()
		s.resize()
		s.Acquire()
	}

}

func (s *Semaphore) Release() {
	s.RLock()
	defer s.RUnlock()
	select {
	case <-s.semOverflow:
		break
	default:
		select {
		case <-s.sem:
		default:
			panic("Released more then acquired!")
		}

	}

}
