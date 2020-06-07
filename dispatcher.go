package gopd

import (
	"container/heap"
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

type Dispatcher struct {
	pq              priorityQueue
	maxMessages     int
	parallelization int

	idleWorkers   []Worker
	activeWorkers map[uuid.UUID]Worker

	egress    chan interface{}
	ingest    chan ScheduledMessage
	processed chan uuid.UUID
	shutdown  chan error
	done      chan context.Context
}

func NewDispatcher(config DispatcherConfig) *Dispatcher {
	queue := make(priorityQueue, 0)
	heap.Init(&queue)

	return &Dispatcher{
		pq:              queue,
		maxMessages:     config.MaxMessages,
		parallelization: config.Parallelization,

		activeWorkers: make(map[uuid.UUID]Worker),

		egress:    make(chan interface{}, config.EgressBufferSize),
		ingest:    make(chan ScheduledMessage, config.IngestBufferSize),
		processed: make(chan uuid.UUID),
		shutdown:  make(chan error),
		done:      make(chan context.Context),
	}
}

func (d *Dispatcher) Start() {
	for i := 0; i < d.parallelization; i++ {
		workerProcessor := NewWorker(uuid.New(), d.processed, d.egress)
		d.idleWorkers = append(d.idleWorkers, workerProcessor)

		go workerProcessor.Process()
	}

	go d.ingestProcess()
	go d.completedProcess()
}

func (d *Dispatcher) Shutdown(ctx context.Context) error {
	d.done <- ctx
	close(d.ingest)

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

func (d *Dispatcher) IngestChannel() chan<- ScheduledMessage {
	return d.ingest
}

func (d *Dispatcher) DispatchChannel() <-chan interface{} {
	return d.egress
}

func (d *Dispatcher) ingestProcess() {
	var lowestActiveWorker Worker

	for {
		if d.pq.Len() >= d.maxMessages {
			continue
		}

		select {
		case ctx, ok := <-d.done:
			if !ok {
				select {
				case <-ctx.Done():
					if len(d.ingest) > 0 {
						d.shutdown <- errors.New("failed to drain ingest channel within deadline")
					}
					if d.pq.Len() > 0 {
						d.shutdown <- errors.New("failed to drain scheduled messages within deadline")
					}

					close(d.shutdown)
					return
				}
			}
		case msg := <-d.ingest:
			if len(d.idleWorkers) > 0 {
				workerProcessor := d.idleWorkers[0]
				d.idleWorkers = d.idleWorkers[1:]
				d.activeWorkers[workerProcessor.Id] = workerProcessor

				workerProcessor.IngestChannel() <- msg

				// set the lowest worker
				if workerProcessor.CurrentMessage.At.After(lowestActiveWorker.CurrentMessage.At) {
					lowestActiveWorker = workerProcessor
				}
			} else if msg.At.Before(lowestActiveWorker.CurrentMessage.At) {
				lowestActiveWorker.IngestChannel() <- msg
			} else {
				heap.Push(&d.pq, msg)
			}
		}
	}
}

func (d *Dispatcher) completedProcess() {
	for {
		select {
		case ctx, ok := <-d.done:
			if !ok {
				select {
				case <-ctx.Done():
					if len(d.ingest) > 0 {
						d.shutdown <- errors.New("failed to drain ingest channel within deadline")
					}
					if d.pq.Len() > 0 {
						d.shutdown <- errors.New("failed to drain scheduled messages within deadline")
					}

					close(d.shutdown)
					return
				}
			}
		case workerId := <-d.processed:
			workerProcessor := d.activeWorkers[workerId]

			if d.pq.Len() > 0 {
				msg, ok := heap.Pop(&d.pq).(ScheduledMessage)
				if !ok {
					// todo
				}
				workerProcessor.IngestChannel() <- msg
			} else {
				delete(d.activeWorkers, workerId)
				d.idleWorkers = append(d.idleWorkers, workerProcessor)
			}
		}
	}
}
