package gosd

import (
	"time"
)

type delayer[T any] interface {
	stop(drain bool)
	wait(msg *ScheduledMessage[T])
	available() bool
}

type delayState int

const (
	idle delayState = iota
	waiting
)

type delay[T any] struct {
	state delayState

	idleChannel   chan<- bool
	egressChannel chan<- T
	cancelChannel chan bool
}

func newDelay[T any](egressChannel chan<- T, idleChannel chan<- bool) *delay[T] {
	return &delay[T]{
		idleChannel:   idleChannel,
		egressChannel: egressChannel,
		cancelChannel: make(chan bool, 1),
	}
}

// stop sends a cancel signal to the current timer process.
func (d *delay[T]) stop(drain bool) {
	if d.state == waiting {
		d.cancelChannel <- drain
	}
}

// wait will create a timer based on the time from `msg.At` and dispatch the message to the egress channel asynchronously.
func (d *delay[T]) wait(msg *ScheduledMessage[T]) {
	d.state = waiting
	curTimer := time.NewTimer(time.Until(msg.At))

	go func() {
		for {
			select {
			case drain, ok := <-d.cancelChannel:
				if ok {
					curTimer.Stop()
					if drain {
						d.egressChannel <- msg.Message
					}
					d.state = idle
					d.idleChannel <- true
				}
				return
			case <-curTimer.C:
				d.egressChannel <- msg.Message
				d.state = idle
				d.idleChannel <- true
				return
			}
		}
	}()
}

// available returns whether the delay is able to accept a new message to wait on.
func (d *delay[T]) available() bool {
	return d.state == idle
}
