package gopd

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

type Dispatcher struct {
	syncPq            SynchronousHeap
	maxMessages       int
	parallelization   int
	sleepWorkersCount int

	lowestActiveWorker *Worker

	requeue     chan ScheduledMessage
	idleWorkers chan *Worker
	egress      chan interface{}
	ingress     chan ScheduledMessage
	processed   chan uuid.UUID
	shutdown    chan error
	done        chan context.Context
}

func NewDispatcher(config DispatcherConfig) *Dispatcher {
	return &Dispatcher{
		syncPq:            NewSynchronousHeap(make(priorityQueue, 0)),
		maxMessages:       config.MaxMessages,
		parallelization:   config.Parallelization,
		sleepWorkersCount: config.SleepWorkersCount,

		requeue:     make(chan ScheduledMessage),
		idleWorkers: make(chan *Worker, config.SleepWorkersCount),
		egress:      make(chan interface{}, config.EgressBufferSize),
		ingress:     make(chan ScheduledMessage, config.IngressBufferSize),
		shutdown:    make(chan error),
		done:        make(chan context.Context),
	}
}

func (d *Dispatcher) Start() {
	for i := 0; i < d.sleepWorkersCount; i++ {
		workerProcessor := NewWorker(uuid.New(), d.egress, d.requeue, d.idleWorkers)
		d.idleWorkers <- &workerProcessor

		go workerProcessor.Process()
	}

	go d.process()
}

func (d *Dispatcher) Shutdown(ctx context.Context) error {
	d.done <- ctx
	close(d.ingress)

	var err error
	for {
		newErr, ok := <-d.shutdown
		if !ok {
			return err
		}

		if newErr != nil {
			err = fmt.Errorf("%w", newErr)
		}
	}
}

func (d *Dispatcher) IngressChannel() chan<- ScheduledMessage {
	return d.ingress
}

func (d *Dispatcher) DispatchChannel() <-chan interface{} {
	return d.egress
}

func (d *Dispatcher) process() {
	for {
		if len(d.idleWorkers) > 0 && d.syncPq.Len() > 0 {
			worker := <-d.idleWorkers

			msg, ok := d.syncPq.Pop().(ScheduledMessage)
			if !ok {
				// todo
			}

			worker.IngestChannel() <- msg
		}

		if d.syncPq.Len() >= d.maxMessages {
			continue
		}

		select {
		case ctx, ok := <-d.done:
			if !ok {
				select {
				case <-ctx.Done():
					if len(d.ingress) > 0 {
						d.shutdown <- errors.New("failed to drain ingress channel within deadline")
					}
					if d.syncPq.Len() > 0 {
						d.shutdown <- errors.New("failed to drain scheduled messages within deadline")
					}

					close(d.shutdown)
					return
				}
			}
		case msg := <-d.requeue:
			d.syncPq.Push(msg)
		case msg := <-d.ingress:
			if d.lowestActiveWorker == nil && len(d.idleWorkers) > 0 {
				workerProcessor := <-d.idleWorkers
				d.lowestActiveWorker = workerProcessor
				workerProcessor.IngestChannel() <- msg
			} else if d.lowestActiveWorker != nil && msg.At.Before(d.lowestActiveWorker.CurrentMessage.At) {
				d.lowestActiveWorker.IngestChannel() <- msg
			} else {
				if len(d.idleWorkers) > 0 {
					workerProcessor := <-d.idleWorkers
					workerProcessor.IngestChannel() <- msg
				} else {
					d.syncPq.Push(msg)
				}
			}
		}
	}
}
