package gopd

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Worker struct {
	Id             uuid.UUID
	CurrentMessage ScheduledMessage

	egress    chan<- interface{}
	ingest    chan ScheduledMessage
	processed chan<- uuid.UUID
	done      chan context.Context
}

func NewWorker(id uuid.UUID, processed chan<- uuid.UUID, egress chan<- interface{}) Worker {
	return Worker{
		Id: id,

		egress:    egress,
		ingest:    make(chan ScheduledMessage),
		processed: processed,
		done:      make(chan context.Context),
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
				w.processed <- w.Id
			}
		}
	}
}
