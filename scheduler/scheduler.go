package scheduler

import (
	"errors"
	"sync"

	"github.com/hashicorp/go-hclog"
)

type tuple = [2]interface{}

type SchedulerOptions struct {
	MaxWorkers  int
	Interactive bool
}

type Scheduler[T comparable] struct {
	WorkerState    chan tuple
	workers        []worker[T]
	workerPool     chan *worker[T]
	quit           bool
	handler        func(T)
	inputQueue     []T
	inputQueueLock sync.Mutex
	opts           SchedulerOptions
}

func NewScheduler[T comparable](opts SchedulerOptions) *Scheduler[T] {
	return &Scheduler[T]{
		WorkerState: make(chan tuple, opts.MaxWorkers),
		workerPool:  make(chan *worker[T], opts.MaxWorkers),
		opts:        opts,
	}
}

func (s *Scheduler[T]) WithHandler(handler func(T)) *Scheduler[T] {
	s.handler = handler

	for i := 0; i < s.opts.MaxWorkers; i++ {
		s.workers = append(s.workers,
			newWorker(i,
				s.handler,
				s.opts.Interactive,
				s.workerPool,
				s.WorkerState))
	}

	return s
}

func (s *Scheduler[T]) Dispatch(tasks []T) {
	for _, t := range tasks {
		s.enqueue(t)
	}
}

func (s *Scheduler[T]) Start() {
	if s.handler == nil {
		err := errors.New("scheduler missing handler")
		hclog.Default().Error(err.Error())
		panic(err)
	}

	for _, w := range s.workers {
		w.start()
	}

	go s.run()
}

func (s *Scheduler[T]) Stop() {
	s.quit = true

	var wg sync.WaitGroup
	for _, w := range s.workers {
		wg.Add(1)
		go func(w worker[T]) {
			defer wg.Done()
			w.cha.quit <- true
		}(w)
	}

	wg.Wait()
}

func (s *Scheduler[T]) run() {
	for {
		if s.quit {
			return
		}

		if len(s.inputQueue) <= 0 {
			continue
		}

		t := s.dequeue()
		worker := <-s.workerPool
		hclog.Default().Trace("scheduler got worker", "id", worker.id)
		worker.cha.tasks <- t
	}
}

func (s *Scheduler[T]) enqueue(t T) {
	s.inputQueue = append(s.inputQueue, t)
}

func (s *Scheduler[T]) dequeue() T {
	s.inputQueueLock.Lock()
	defer s.inputQueueLock.Unlock()
	t := s.inputQueue[0]
	s.inputQueue = s.inputQueue[1:]
	return t
}
