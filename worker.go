package gopd

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Worker struct {
	Id             uuid.UUID
	CurrentMessage ScheduledMessage

	requeue     chan<- ScheduledMessage
	idleWorkers chan<- *Worker
	egress      chan<- interface{}
	ingest      chan ScheduledMessage
	done        chan context.Context
}

func NewWorker(id uuid.UUID, egress chan<- interface{}, requeue chan<- ScheduledMessage, idleWorkers chan<- *Worker) Worker {
	return Worker{
		Id: id,

		requeue:     requeue,
		idleWorkers: idleWorkers,
		egress:      egress,
		ingest:      make(chan ScheduledMessage),
		done:        make(chan context.Context),
	}
}

func (w *Worker) IngestChannel() chan<- ScheduledMessage {
	return w.ingest
}

func (w *Worker) Process() {
	var (
		curTimer *time.Timer
	)

	for {
		select {
		case _, ok := <-w.done:
			if !ok {
				// todo
			}
		case msg := <-w.ingest:
			if curTimer != nil {
				curTimer.Stop()
				w.requeue <- w.CurrentMessage
			}

			duration := msg.At.Sub(time.Now())
			curTimer = time.NewTimer(duration)
			w.CurrentMessage = msg
		default:
			if curTimer != nil {
				<-curTimer.C
				w.egress <- w.CurrentMessage
				curTimer.Stop()
				curTimer = nil
				w.idleWorkers <- w
			}
		}
	}
}
