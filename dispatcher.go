package gosd

import (
	"container/heap"
	"context"
	"errors"
	"sync"
)

// dispatcherState represents state for a Dispatcher.
type dispatcherState int

const (
	paused dispatcherState = iota
	processing
	shutdown
	shutdownAndDrain
)

// Dispatcher processes the ingress and dispatching of scheduled messages.
type Dispatcher[T any] struct {
	state       dispatcherState
	maxMessages int

	pq          priorityQueue[T]
	nextMessage *ScheduledMessage[T]
	delayer     delayer[T]

	delayerIdleChannel chan bool
	dispatchChannel    chan T
	ingressChannel     chan *ScheduledMessage[T]
	shutdown           chan error
	stopProcess        chan bool

	mutex *sync.Mutex
}

// NewDispatcher creates a new instance of a Dispatcher.
func NewDispatcher[T any](config *DispatcherConfig) (*Dispatcher[T], error) {
	if config.MaxMessages <= 0 {
		return nil, errors.New("MaxMessages should be greater than 0")
	}

	newIdleChannel := make(chan bool, 1)
	newDispatchChannel := make(chan T, config.DispatchChannelSize)
	newPq := priorityQueue[T]{
		items:         make([]*item[T], 0),
		maintainOrder: config.GuaranteeOrder,
	}

	heap.Init(&newPq)
	return &Dispatcher[T]{
		pq:                 newPq,
		maxMessages:        config.MaxMessages,
		delayer:            newDelay[T](newDispatchChannel, newIdleChannel),
		delayerIdleChannel: newIdleChannel,
		dispatchChannel:    newDispatchChannel,
		ingressChannel:     make(chan *ScheduledMessage[T], config.IngressChannelSize),
		shutdown:           make(chan error),
		stopProcess:        make(chan bool),
		mutex:              &sync.Mutex{},
	}, nil
}

// Shutdown will attempt to shutdown the Dispatcher within the context deadline, otherwise terminating the process
// ungracefully.
//
// If drainImmediately is true, then all messages will be dispatched immediately regardless of the schedule set. Order
// can be lost if new messages are still being ingested.
func (d *Dispatcher[T]) Shutdown(ctx context.Context, drainImmediately bool) error {
	if d.state == shutdown || d.state == shutdownAndDrain {
		return errors.New("shutdown has already happened")
	}

	d.mutex.Lock()
	defer d.mutex.Unlock()

	// if paused, resume the process in order to drain messages
	if d.state == paused {
		d.delayer.wait(d.nextMessage)
		go d.process()
	}

	if drainImmediately {
		d.state = shutdownAndDrain
	} else {
		d.state = shutdown
	}

	// block new messages and let the channel drain
	close(d.ingressChannel)

	for {
		select {
		case <-ctx.Done():
			// forcefully kill the process regardless of messages left
			close(d.stopProcess)
			close(d.dispatchChannel)
			return errors.New("failed to gracefully drain and shutdown dispatcher within deadline")
		default:
			// wait for the ingress channel and heap to drain
			if len(d.ingressChannel) == 0 && d.pq.Len() == 0 && d.delayer.available() {
				close(d.stopProcess)
				close(d.dispatchChannel)
				return nil
			}
		}
	}
}

// Start initializes the processing of scheduled messages and blocks.
func (d *Dispatcher[T]) Start() error {
	d.mutex.Lock()
	if d.state == shutdown || d.state == shutdownAndDrain {
		return errors.New("dispatcher is already running and shutting/shut down")
	} else if d.state == processing {
		return errors.New("dispatcher is already running")
	}

	d.state = processing
	d.mutex.Unlock()
	d.process()
	return nil
}

// Pause updates the state of the Dispatcher to stop processing messages and will close the main process loop.
func (d *Dispatcher[T]) Pause() error {
	d.mutex.Lock()
	if d.state == shutdown || d.state == shutdownAndDrain {
		return errors.New("dispatcher is shutting/shut down and cannot be paused")
	} else if d.state == paused {
		return errors.New("dispatcher is already paused")
	}

	d.state = paused
	d.stopProcess <- true
	d.delayer.stop(false)
	d.mutex.Unlock()
	return nil
}

// Resume updates the state of the Dispatcher to start processing messages and starts the timer for the last message
// being processed and blocks.
func (d *Dispatcher[T]) Resume() error {
	d.mutex.Lock()
	if d.state == shutdown || d.state == shutdownAndDrain {
		return errors.New("dispatcher is shutting/shut down")
	} else if d.state == processing {
		return errors.New("dispatcher is already running")
	}

	d.state = processing
	if d.nextMessage != nil {
		d.delayer.wait(d.nextMessage)
	}
	d.mutex.Unlock()
	d.process()
	return nil
}

// process handles the processing of scheduled messages.
func (d *Dispatcher[T]) process() {
	for {
		select {
		case <-d.stopProcess:
			return
		default:
			// check if the state has changed to `shutdownAndDrain`
			d.handleShutdownAndDrain()

			// check pq for the next message to wait on and continue if not full
			if cont := d.handlePriorityQueue(); !cont {
				continue
			}

			// check the ingress channel for new messages
			d.handleIngress()
		}
	}
}

// handleShutdown drains the heap.
func (d *Dispatcher[T]) handleShutdownAndDrain() {
	if d.state == shutdownAndDrain {
		d.delayer.stop(true)
		if len(d.delayerIdleChannel) > 0 {
			<-d.delayerIdleChannel
			d.drainHeap()
		}
	}
}

// handlePriorityQueue checks whether the heap is full and will Pop the next message if present and when the delayer is
// idle.
func (d *Dispatcher[T]) handlePriorityQueue() (cont bool) {
	// check if we've exceeded the maximum messages to store in the heap
	if d.pq.Len() >= d.maxMessages {
		if len(d.delayerIdleChannel) > 0 {
			<-d.delayerIdleChannel
			d.waitNextMessage()
		}
		// skip ingest to prevent heap from exceeding MaxMessages
		return false
	} else if d.pq.Len() > 0 && len(d.delayerIdleChannel) > 0 {
		<-d.delayerIdleChannel
		d.waitNextMessage()
	}
	return true
}

// handleIngress checks for new messages off the ingress channel and will either dispatch if `shutdownAndDrain`, replace
// the current delayer message or add to the heap.
func (d *Dispatcher[T]) handleIngress() {
	if len(d.ingressChannel) > 0 {
		if msg, ok := <-d.ingressChannel; ok {
			if d.state == shutdownAndDrain {
				// dispatch the new message immediately
				d.dispatchChannel <- msg.Message
			} else if d.nextMessage != nil && msg.At.Before(d.nextMessage.At) {
				heap.Push(&d.pq, d.nextMessage)
				d.nextMessage = msg
				d.delayer.stop(false)
				<-d.delayerIdleChannel
				d.delayer.wait(msg)
			} else if d.nextMessage == nil {
				d.nextMessage = msg
				d.delayer.wait(msg)
			} else {
				heap.Push(&d.pq, msg)
			}
		}
	}
}

func (d *Dispatcher[T]) waitNextMessage() {
	msg := heap.Pop(&d.pq).(*ScheduledMessage[T])
	d.nextMessage = msg
	d.delayer.wait(msg)
}

func (d *Dispatcher[T]) drainHeap() {
	for d.pq.Len() > 0 {
		msg := heap.Pop(&d.pq).(*ScheduledMessage[T])
		// dispatch the message immediately
		d.dispatchChannel <- msg.Message
	}
}

// IngressChannel returns the send-only channel of type `ScheduledMessage`.
func (d *Dispatcher[T]) IngressChannel() chan<- *ScheduledMessage[T] {
	return d.ingressChannel
}

// DispatchChannel returns a receive-only channel of type `T`.
func (d *Dispatcher[T]) DispatchChannel() <-chan T {
	return d.dispatchChannel
}
