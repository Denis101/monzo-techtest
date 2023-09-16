package scheduler

import (
	"github.com/hashicorp/go-hclog"
)

type worker[T comparable] struct {
	id          int
	handler     func(T)
	reportState bool
	cha         workerChannels[T]
}

type workerChannels[T comparable] struct {
	pool  chan *worker[T]
	state chan tuple
	tasks chan T
	quit  chan bool
}

func newWorker[T comparable](
	id int,
	handler func(T),
	reportState bool,
	pool chan *worker[T],
	state chan tuple) worker[T] {
	return worker[T]{
		id:          id,
		handler:     handler,
		reportState: reportState,
		cha: workerChannels[T]{
			pool:  pool,
			state: state,
			tasks: make(chan T),
			quit:  make(chan bool),
		},
	}
}

func (w worker[T]) start() {
	go func() {
		for {
			hclog.Default().Trace("worker waiting", "id", w.id)
			w.cha.pool <- &w

			select {
			case task := <-w.cha.tasks:
				hclog.Default().Trace("worker start task", "id", w.id)
				if w.reportState {
					w.cha.state <- tuple{w.id, task}
				}

				w.handler(task)
				hclog.Default().Trace("worker end task", "id", w.id)
			case <-w.cha.quit:
				close(w.cha.tasks)
				return
			}
		}
	}()
}
