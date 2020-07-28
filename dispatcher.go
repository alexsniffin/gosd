package gopd

import (
	"container/heap"
	"context"
	"errors"
)

// dispatcherState represents state for a Dispatcher
type dispatcherState int

const (
	Paused dispatcherState = iota
	Processing
	Shutdown
	ShutdownAndDrain
)

// Dispatcher processes the ingress and dispatching of scheduled messages
type Dispatcher struct {
	state          dispatcherState
	maxMessages    int
	guaranteeOrder bool

	pq          priorityQueue
	nextMessage *ScheduledMessage
	delayer     delayer

	delayerIdleChannel chan bool
	dispatchChannel    chan interface{}
	ingressChannel     chan *ScheduledMessage
	shutdown           chan error
	stopProcess        chan bool
}

// NewDispatcher creates a new instance of a Dispatcher
func NewDispatcher(config *DispatcherConfig) (*Dispatcher, error) {
	if config.MaxMessages <= 0 {
		return nil, errors.New("MaxMessages should be greater than 0")
	}

	newIdleChannel := make(chan bool, 1)
	newDispatchChannel := make(chan interface{}, config.DispatchChannelSize)
	newPq := priorityQueue{
		items:         make([]*item, 0),
		maintainOrder: config.GuaranteeOrder,
	}

	heap.Init(&newPq)
	return &Dispatcher{
		pq:                 newPq,
		maxMessages:        config.MaxMessages,
		guaranteeOrder:     config.GuaranteeOrder,
		delayer:            newDelay(!config.GuaranteeOrder, newDispatchChannel, newIdleChannel),
		delayerIdleChannel: newIdleChannel,
		dispatchChannel:    newDispatchChannel,
		ingressChannel:     make(chan *ScheduledMessage, config.IngressChannelSize),
		shutdown:           make(chan error),
		stopProcess:        make(chan bool),
	}, nil
}

// Shutdown will attempt to shutdown the Dispatcher within the context deadline, otherwise terminating the process
// ungracefully
//
// If drainImmediately is true, then all messages will be dispatched immediately regardless of the schedule set
func (d *Dispatcher) Shutdown(ctx context.Context, drainImmediately bool) error {
	// if paused, resume the process in order to drain messages
	if d.state == Paused {
		d.delayer.wait(d.nextMessage)
		go d.process()
	}

	if drainImmediately {
		d.state = ShutdownAndDrain
	} else {
		d.state = Shutdown
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

// Start initializes the processing of scheduled messages
func (d *Dispatcher) Start() error {
	if d.state == Shutdown || d.state == ShutdownAndDrain {
		return errors.New("dispatcher is already running and shutting down")
	} else if d.state == Processing {
		return errors.New("dispatcher is already running")
	}

	d.state = Processing
	d.process()
	return nil
}

// Pause updates the state of the Dispatcher to stop processing messages
func (d *Dispatcher) Pause() error {
	if d.state == Shutdown || d.state == ShutdownAndDrain {
		return errors.New("dispatcher is shutting down and cannot be paused")
	} else if d.state == Paused {
		return errors.New("dispatcher is already paused")
	}

	d.state = Paused
	d.stopProcess <- true
	d.delayer.stop()
	return nil
}

// Resume updates the state of the Dispatcher to start processing messages and starts the timer for the last message
// being processed
func (d *Dispatcher) Resume() error {
	if d.state == Shutdown || d.state == ShutdownAndDrain {
		return errors.New("dispatcher is shutting down")
	} else if d.state == Processing {
		return errors.New("dispatcher is already running")
	}

	d.state = Processing
	if d.nextMessage != nil {
		d.delayer.wait(d.nextMessage)
	}
	d.process()
	return nil
}

// process handles the processing of scheduled messages
func (d *Dispatcher) process() {
	for {
		select {
		case <-d.stopProcess:
			return
		default:
			// drain the heap
			if d.state == ShutdownAndDrain {
				d.drainHeap()
			}

			// check if we've exceeded the maximum messages to store in the heap
			if d.pq.Len() >= d.maxMessages {
				if !d.guaranteeOrder && len(d.delayerIdleChannel) > 0 {
					<-d.delayerIdleChannel
					d.waitNextMessage()
				} else if d.delayer.available() {
					d.waitNextMessage()
				}
				// skip ingest to prevent heap from exceeding MaxMessages
				continue
			} else if d.pq.Len() > 0 {
				if !d.guaranteeOrder && len(d.delayerIdleChannel) > 0 {
					<-d.delayerIdleChannel
					d.waitNextMessage()
				} else if d.delayer.available() {
					d.waitNextMessage()
				}
			}

			if len(d.ingressChannel) > 0 {
				if msg, ok := <-d.ingressChannel; ok {
					if d.state == ShutdownAndDrain {
						// dispatch the new message immediately
						d.dispatchChannel <- msg
					} else if d.nextMessage != nil && msg.At.Before(d.nextMessage.At) {
						heap.Push(&d.pq, d.nextMessage)
						d.nextMessage = msg
						d.delayer.stop()
						if !d.guaranteeOrder {
							<-d.delayerIdleChannel
						}
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
	}
}

func (d *Dispatcher) waitNextMessage() {
	msg := heap.Pop(&d.pq).(*ScheduledMessage)
	d.nextMessage = msg
	d.delayer.wait(msg)
}

func (d *Dispatcher) drainHeap() {
	for d.pq.Len() > 0 {
		msg := heap.Pop(&d.pq).(*ScheduledMessage)
		// dispatch the message immediately
		d.dispatchChannel <- msg
	}
}

// IngressChannel returns the send-only channel of type `ScheduledMessage`
func (d *Dispatcher) IngressChannel() chan<- *ScheduledMessage {
	return d.ingressChannel
}

// DispatchChannel returns a receive-only channel of type `interface{}`
func (d *Dispatcher) DispatchChannel() <-chan interface{} {
	return d.dispatchChannel
}
