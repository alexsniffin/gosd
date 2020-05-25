package gopd

import (
	"container/heap"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/alexsniffin/gopd/internal/pq"
	"github.com/alexsniffin/gopd/internal/worker"
)

type ScheduledMessage struct {
	At      time.Time
	Message interface{}
}

type Dispatcher struct {
	pq              pq.PriorityQueue
	parallelization int

	egress   chan interface{}
	ingest   chan ScheduledMessage
	shutdown chan error
	done     chan context.Context
}

func NewDispatcher(parallelization, ingestBufferSize, egressBufferSize int) *Dispatcher {
	queue := make(pq.PriorityQueue, 0)
	heap.Init(&queue)

	return &Dispatcher{
		pq:              queue,
		parallelization: parallelization,

		egress:   make(chan interface{}, egressBufferSize),
		ingest:   make(chan ScheduledMessage, ingestBufferSize),
		shutdown: make(chan error),
		done:     make(chan context.Context),
	}
}

func (d *Dispatcher) Start() {
	go d.scheduler()
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

func (d *Dispatcher) scheduler() {
	processed := make(chan uuid.UUID)
	idleWorkers := make([]worker.Processor, d.parallelization)
	activeWorkers := make(map[uuid.UUID]worker.Processor)

	for i := 0; i < d.parallelization; i++ {
		workerProcessor := worker.NewProcessor(uuid.New(), processed, d.egress)
		idleWorkers = append(idleWorkers, workerProcessor)

		go workerProcessor.Process()
	}

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
		case workerId := <-processed:
			workerProcessor := activeWorkers[workerId]

			if d.pq.Len() > 0 {
				msg := d.pq.Pop().(ScheduledMessage)
				workerProcessor.IngestChannel() <- msg
			} else {
				delete(activeWorkers, workerId)
				idleWorkers = append(idleWorkers, workerProcessor)
			}
		case msg := <-d.ingest:
			if len(idleWorkers) > 0 {
				workerProcessor := idleWorkers[0]
				idleWorkers = idleWorkers[1:]
				activeWorkers[workerProcessor.Id] = workerProcessor

				workerProcessor.IngestChannel() <- msg
			}
			// todo swap lowest active worker and add to queue
		}
	}
}
