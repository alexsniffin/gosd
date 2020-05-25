package worker

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/alexsniffin/gopd"
)

type Processor struct {
	Id              uuid.UUID
	MessageDeadline time.Time

	egress    chan<- interface{}
	ingest    chan gopd.ScheduledMessage
	processed chan<- uuid.UUID
	done      chan context.Context
}

func NewProcessor(id uuid.UUID, processed chan<- uuid.UUID, egress chan<- interface{}) Processor {
	return Processor{
		Id: id,

		egress:    egress,
		ingest:    make(chan gopd.ScheduledMessage),
		processed: processed,
		done:      make(chan context.Context),
	}
}

func (p *Processor) IngestChannel() chan<- gopd.ScheduledMessage {
	return p.ingest
}

func (p *Processor) Process() {
	var (
		curTimer   *time.Timer
		curMessage interface{}
	)

	for {
		select {
		case _, ok := <-p.done:
			if !ok {
				// todo
			}
		case msg := <-p.ingest:
			if curTimer != nil {
				curTimer.Stop()
			}

			duration := msg.At.Sub(time.Now())
			curTimer = time.NewTimer(duration)
			curMessage = msg.Message
		case <-curTimer.C:
			p.egress <- curMessage
			curTimer.Stop()
		}
	}
}
