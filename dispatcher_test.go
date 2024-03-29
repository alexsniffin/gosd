package gosd

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"
)

func skipCI(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}
}

type fakeDelayer[T any] struct {
	stopCalled        chan bool
	waitCalled        chan *ScheduledMessage[T]
	availableCalled   chan bool
	availableResponse bool
}

func (fd *fakeDelayer[T]) stop(drain bool) {
	if fd.stopCalled != nil {
		fd.stopCalled <- true
	}
}

func (fd *fakeDelayer[T]) wait(msg *ScheduledMessage[T]) {
	if fd.waitCalled != nil {
		fd.waitCalled <- msg
	}
}

func (fd *fakeDelayer[T]) available() bool {
	if fd.availableCalled != nil {
		fd.availableCalled <- true
	}
	return fd.availableResponse
}

func TestDispatcher_Start(t *testing.T) {
	t.Parallel()

	type fields struct {
		state           dispatcherState
		pq              priorityQueue[any]
		maxMessages     int
		nextMessage     *ScheduledMessage[any]
		delayer         delayer[any]
		dispatchChannel chan interface{}
		ingressChannel  chan *ScheduledMessage[any]
		shutdown        chan error
		stopProcess     chan bool
	}
	tests := []struct {
		name      string
		fields    fields
		wantState dispatcherState
		wantErr   bool
	}{
		{"pausedState", fields{state: paused, maxMessages: 1, stopProcess: make(chan bool)}, processing, false},
		{"processingState", fields{state: processing}, processing, true},
		{"shutdownState", fields{state: shutdown}, shutdown, true},
		{"shutdownAndDrainState", fields{state: shutdownAndDrain}, shutdownAndDrain, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Dispatcher[any]{
				state:           tt.fields.state,
				pq:              tt.fields.pq,
				maxMessages:     tt.fields.maxMessages,
				nextMessage:     tt.fields.nextMessage,
				delayer:         tt.fields.delayer,
				dispatchChannel: tt.fields.dispatchChannel,
				ingressChannel:  tt.fields.ingressChannel,
				shutdown:        tt.fields.shutdown,
				stopProcess:     tt.fields.stopProcess,
				mutex:           &sync.Mutex{},
			}
			if d.stopProcess != nil {
				close(d.stopProcess)
			}
			if err := d.Start(); (err != nil) != tt.wantErr {
				t.Errorf("Start() error = %v, wantErr %v", err, tt.wantErr)
			}
			if d.state != tt.wantState {
				t.Errorf("Start() invalid state = %v, wantState %v", d.state, tt.wantState)
			}
		})
	}
}

func TestDispatcher_Pause(t *testing.T) {
	t.Parallel()

	type fields struct {
		state           dispatcherState
		pq              priorityQueue[any]
		maxMessages     int
		nextMessage     *ScheduledMessage[any]
		delayer         delayer[any]
		dispatchChannel chan interface{}
		ingressChannel  chan *ScheduledMessage[any]
		shutdown        chan error
		stopProcess     chan bool
	}
	tests := []struct {
		name           string
		fields         fields
		stopProcessLen int
		wantState      dispatcherState
		wantErr        bool
	}{
		{"processingState", fields{state: processing, delayer: &fakeDelayer[any]{stopCalled: make(chan bool, 1)}, stopProcess: make(chan bool, 1)}, 1, paused, false},
		{"pausedState", fields{state: paused}, 0, paused, true},
		{"shutdownState", fields{state: shutdown}, 0, shutdown, true},
		{"shutdownAndDrainState", fields{state: shutdownAndDrain}, 0, shutdownAndDrain, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Dispatcher[any]{
				state:           tt.fields.state,
				pq:              tt.fields.pq,
				maxMessages:     tt.fields.maxMessages,
				nextMessage:     tt.fields.nextMessage,
				delayer:         tt.fields.delayer,
				dispatchChannel: tt.fields.dispatchChannel,
				ingressChannel:  tt.fields.ingressChannel,
				shutdown:        tt.fields.shutdown,
				stopProcess:     tt.fields.stopProcess,
				mutex:           &sync.Mutex{},
			}
			if err := d.Pause(); (err != nil) != tt.wantErr {
				t.Errorf("Pause() error = %v, wantErr %v", err, tt.wantErr)
			}
			if d.state != tt.wantState {
				t.Errorf("Pause() invalid state = %v, wantState %v", d.state, tt.wantState)
			}
			if len(d.stopProcess) != tt.stopProcessLen {
				t.Errorf("Pause() invalid stopProcess length = %d, want = %d", len(d.stopProcess), tt.stopProcessLen)
			}
			if fd, ok := tt.fields.delayer.(*fakeDelayer[any]); ok {
				if len(fd.stopCalled) == 0 {
					t.Error("Pause() delayer.stop() call expected")
				}
			}
		})
	}
}

func TestDispatcher_Resume(t *testing.T) {
	t.Parallel()

	type fields struct {
		state           dispatcherState
		pq              priorityQueue[any]
		maxMessages     int
		nextMessage     *ScheduledMessage[any]
		delayer         delayer[any]
		dispatchChannel chan interface{}
		ingressChannel  chan *ScheduledMessage[any]
		shutdown        chan error
		stopProcess     chan bool
	}
	tests := []struct {
		name      string
		fields    fields
		wantState dispatcherState
		wantErr   bool
	}{
		{"pausedState", fields{
			state: paused,
			pq: priorityQueue[any]{
				items:         make([]*item[any], 0),
				maintainOrder: false,
			},
			maxMessages: 1,
			nextMessage: &ScheduledMessage[any]{},
			delayer:     &fakeDelayer[any]{waitCalled: make(chan *ScheduledMessage[any], 1), availableResponse: false},
			stopProcess: make(chan bool),
		}, processing, false},
		{"pausedStateNilNextMessage", fields{
			state: paused,
			pq: priorityQueue[any]{
				items:         make([]*item[any], 0),
				maintainOrder: false,
			},
			maxMessages: 1,
			stopProcess: make(chan bool),
		}, processing, false},
		{"processingState", fields{state: processing}, processing, true},
		{"shutdownState", fields{state: shutdown}, shutdown, true},
		{"shutdownAndDrainState", fields{state: shutdownAndDrain}, shutdownAndDrain, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Dispatcher[any]{
				state:           tt.fields.state,
				pq:              tt.fields.pq,
				maxMessages:     tt.fields.maxMessages,
				nextMessage:     tt.fields.nextMessage,
				delayer:         tt.fields.delayer,
				dispatchChannel: tt.fields.dispatchChannel,
				ingressChannel:  tt.fields.ingressChannel,
				shutdown:        tt.fields.shutdown,
				stopProcess:     tt.fields.stopProcess,
				mutex:           &sync.Mutex{},
			}
			if d.stopProcess != nil {
				close(d.stopProcess)
			}
			if err := d.Resume(); (err != nil) != tt.wantErr {
				t.Errorf("Resume() error = %v, wantErr %v", err, tt.wantErr)
			}
			if d.state != tt.wantState {
				t.Errorf("Resume() invalid state = %v, wantState %v", d.state, tt.wantState)
			}
			if fd, ok := tt.fields.delayer.(*fakeDelayer[any]); ok {
				if len(fd.waitCalled) == 0 {
					t.Error("Resume() delayer.wait() call expected")
				}
			}
		})
	}
}

func TestDispatcher_process(t *testing.T) {
	t.Parallel()

	type fields struct {
		state              dispatcherState
		pq                 priorityQueue[any]
		maxMessages        int
		nextMessage        *ScheduledMessage[any]
		delayer            delayer[any]
		delayerIdleChannel chan bool
		dispatchChannel    chan interface{}
		ingressChannel     chan *ScheduledMessage[any]
		shutdown           chan error
		stopProcess        chan bool
	}
	tests := []struct {
		name            string
		fields          fields
		customAssertion func(*Dispatcher[any])
	}{
		{"pqMaxMessageSize", fields{
			state: processing,
			delayer: &fakeDelayer[any]{
				availableResponse: true,
				availableCalled:   make(chan bool),
				waitCalled:        make(chan *ScheduledMessage[any], 1)},
			pq: priorityQueue[any]{
				items: []*item[any]{{&ScheduledMessage[any]{}, 0}},
			},
			maxMessages:        1,
			delayerIdleChannel: make(chan bool, 1),
			stopProcess:        make(chan bool)},
			func(d *Dispatcher[any]) {
				if fd, ok := d.delayer.(*fakeDelayer[any]); ok {
					d.delayerIdleChannel <- true
					if _, ok := <-fd.waitCalled; !ok {
						t.Error("process() unexpected close of delayer.wait()")
					}
					fd.waitCalled = nil

					close(d.stopProcess)

					if d.nextMessage == nil {
						t.Error("process() dispatcher nextMessage expected")
					}
				} else {
					t.Error("process() unexpected testing delayer")
				}
			}},
		{"shutdownAndDrain", fields{
			state: shutdownAndDrain,
			delayer: &fakeDelayer[any]{
				stopCalled:        make(chan bool),
				availableResponse: true,
				availableCalled:   make(chan bool),
			},
			pq: priorityQueue[any]{
				items: []*item[any]{{&ScheduledMessage[any]{}, 0}},
			},
			maxMessages:        1,
			dispatchChannel:    make(chan interface{}),
			stopProcess:        make(chan bool),
			delayerIdleChannel: make(chan bool, 1)},
			func(d *Dispatcher[any]) {
				if fd, ok := d.delayer.(*fakeDelayer[any]); ok {
					d.delayerIdleChannel <- true
					if _, ok := <-fd.stopCalled; !ok {
						t.Error("process() unexpected close of delayer.stop()")
					}
					fd.stopCalled = nil

					if _, ok := <-d.dispatchChannel; !ok {
						t.Error("process() message expected from dispatchChannel")
					}

					close(d.stopProcess)

					if d.pq.Len() != 0 {
						t.Errorf("process() unexpected pq length = %d, wanted 0", d.pq.Len())
					}
				} else {
					t.Error("process() unexpected testing delayer")
				}
			}},
		{"ingressChannelNextMessageNil", fields{
			state: processing,
			delayer: &fakeDelayer[any]{
				waitCalled: make(chan *ScheduledMessage[any])},
			pq: priorityQueue[any]{
				items:         make([]*item[any], 0),
				maintainOrder: false,
			},
			maxMessages:    1,
			ingressChannel: make(chan *ScheduledMessage[any], 1),
			stopProcess:    make(chan bool)},
			func(d *Dispatcher[any]) {
				if fd, ok := d.delayer.(*fakeDelayer[any]); ok {
					msg := &ScheduledMessage[any]{}
					d.ingressChannel <- msg
					if _, ok := <-fd.waitCalled; !ok {
						t.Error("process() unexpected close of delayer.wait()")
					}
					fd.waitCalled = nil

					close(d.stopProcess)

					if d.pq.Len() == 1 {
						t.Errorf("process() unexpected pq length = %d, want = 1", d.pq.Len())
					}
				} else {
					t.Error("process() unexpected testing delayer")
				}
			}},
		{"ingressChannelPushHeap", fields{
			state:   processing,
			delayer: &fakeDelayer[any]{},
			pq: priorityQueue[any]{
				items:         make([]*item[any], 0),
				maintainOrder: false,
			},
			maxMessages:    1,
			nextMessage:    &ScheduledMessage[any]{},
			ingressChannel: make(chan *ScheduledMessage[any], 1),
			stopProcess:    make(chan bool)},
			func(d *Dispatcher[any]) {
				if _, ok := d.delayer.(*fakeDelayer[any]); ok {
					msg := &ScheduledMessage[any]{}
					d.ingressChannel <- msg

					close(d.stopProcess)

					if d.nextMessage == msg {
						t.Error("process() unexpected nextMessage")
					}
				} else {
					t.Error("process() unexpected testing delayer")
				}
			}},
		{"ingressChannelReplaceNextMessage", fields{
			state: processing,
			delayer: &fakeDelayer[any]{
				waitCalled: make(chan *ScheduledMessage[any]),
				stopCalled: make(chan bool)},
			pq: priorityQueue[any]{
				items:         make([]*item[any], 0),
				maintainOrder: false,
			},
			maxMessages:        1,
			nextMessage:        &ScheduledMessage[any]{At: time.Now().Add(10 + time.Second)},
			delayerIdleChannel: make(chan bool, 1),
			ingressChannel:     make(chan *ScheduledMessage[any], 1),
			stopProcess:        make(chan bool)},
			func(d *Dispatcher[any]) {
				if fd, ok := d.delayer.(*fakeDelayer[any]); ok {
					msg := &ScheduledMessage[any]{At: time.Now()}
					d.ingressChannel <- msg
					if _, ok := <-fd.stopCalled; !ok {
						t.Error("process() unexpected close of delayer.stop()")
					}
					fd.stopCalled = nil

					d.delayerIdleChannel <- true

					if _, ok := <-fd.waitCalled; !ok {
						t.Error("process() unexpected close of delayer.wait()")
					}
					fd.waitCalled = nil

					close(d.stopProcess)

					if d.nextMessage != msg {
						t.Error("process() unexpected nextMessage")
					}
				} else {
					t.Error("process() unexpected testing delayer")
				}
			}},
		{"ingressChannelShutdownAndDrain", fields{
			state:   shutdownAndDrain,
			delayer: &fakeDelayer[any]{},
			pq: priorityQueue[any]{
				items:         make([]*item[any], 0),
				maintainOrder: false,
			},
			maxMessages:     1,
			ingressChannel:  make(chan *ScheduledMessage[any], 1),
			dispatchChannel: make(chan interface{}),
			stopProcess:     make(chan bool)},
			func(d *Dispatcher[any]) {
				if _, ok := d.delayer.(*fakeDelayer[any]); ok {
					msg := &ScheduledMessage[any]{At: time.Now()}
					d.ingressChannel <- msg

					if _, ok := <-d.dispatchChannel; !ok {
						t.Error("process() message expected from dispatchChannel")
					}

					close(d.stopProcess)
				} else {
					t.Error("process() unexpected testing delayer")
				}
			}},
		{"pqPop", fields{
			state: processing,
			delayer: &fakeDelayer[any]{
				availableResponse: true,
				availableCalled:   make(chan bool),
				waitCalled:        make(chan *ScheduledMessage[any])},
			pq: priorityQueue[any]{
				items: []*item[any]{{&ScheduledMessage[any]{}, 0}},
			},
			maxMessages:        2,
			dispatchChannel:    make(chan interface{}),
			stopProcess:        make(chan bool),
			delayerIdleChannel: make(chan bool, 1)},
			func(d *Dispatcher[any]) {
				if fd, ok := d.delayer.(*fakeDelayer[any]); ok {
					d.delayerIdleChannel <- true
					if _, ok := <-fd.waitCalled; !ok {
						t.Error("process() unexpected close of delayer.wait()")
					}
					fd.waitCalled = nil

					close(d.stopProcess)

					if d.nextMessage == nil {
						t.Error("process() unexpected nil nextMessage")
					}
				} else {
					t.Error("process() unexpected testing delayer")
				}
			}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Dispatcher[any]{
				state:              tt.fields.state,
				pq:                 tt.fields.pq,
				maxMessages:        tt.fields.maxMessages,
				nextMessage:        tt.fields.nextMessage,
				delayer:            tt.fields.delayer,
				delayerIdleChannel: tt.fields.delayerIdleChannel,
				dispatchChannel:    tt.fields.dispatchChannel,
				ingressChannel:     tt.fields.ingressChannel,
				shutdown:           tt.fields.shutdown,
				stopProcess:        tt.fields.stopProcess,
				mutex:              &sync.Mutex{},
			}
			go tt.customAssertion(d)
			d.process()
		})
	}
}

func TestDispatcher_Shutdown(t *testing.T) {
	t.Parallel()

	type fields struct {
		state              dispatcherState
		pq                 priorityQueue[any]
		maxMessages        int
		nextMessage        *ScheduledMessage[any]
		delayer            delayer[any]
		delayerIdleChannel chan bool
		dispatchChannel    chan interface{}
		ingressChannel     chan *ScheduledMessage[any]
		shutdown           chan error
		stopProcess        chan bool
	}
	type args struct {
		ctx              context.Context
		drainImmediately bool
	}
	tests := []struct {
		name            string
		fields          fields
		args            args
		wantErr         bool
		customAssertion func(*Dispatcher[any])
	}{
		{"shutdownWithinDeadline", fields{
			state: processing,
			pq: priorityQueue[any]{
				items:         make([]*item[any], 0),
				maintainOrder: false,
			},
			delayer: &fakeDelayer[any]{
				availableResponse: true,
				availableCalled:   make(chan bool)},
			delayerIdleChannel: make(chan bool),
			ingressChannel:     make(chan *ScheduledMessage[any]),
			dispatchChannel:    make(chan interface{}),
			stopProcess:        make(chan bool)}, args{
			ctx: func() context.Context {
				ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
				return ctx
			}(),
			drainImmediately: false}, false,
			func(d *Dispatcher[any]) {
				if d.state != shutdown {
					t.Errorf("Shutdown() unexpect state = %v, want Shutdown", d.state)
				}
			}},
		{"shutdownWithinDeadlineDrain", fields{
			state: processing,
			pq: priorityQueue[any]{
				items:         make([]*item[any], 0),
				maintainOrder: false,
			},
			delayer: &fakeDelayer[any]{
				availableResponse: true,
				availableCalled:   make(chan bool)},
			delayerIdleChannel: make(chan bool),
			ingressChannel:     make(chan *ScheduledMessage[any]),
			dispatchChannel:    make(chan interface{}),
			stopProcess:        make(chan bool)}, args{
			ctx: func() context.Context {
				ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
				return ctx
			}(),
			drainImmediately: true}, false,
			func(d *Dispatcher[any]) {
				if d.state != shutdownAndDrain {
					t.Errorf("Shutdown() unexpect state = %v, want Shutdown", d.state)
				}

			}},
		{"shutdownNotWithinDeadline", fields{
			state: processing,
			pq: priorityQueue[any]{
				items:         make([]*item[any], 0),
				maintainOrder: false,
			},
			delayer: &fakeDelayer[any]{
				availableResponse: true,
				availableCalled:   make(chan bool)},
			delayerIdleChannel: make(chan bool),
			ingressChannel:     make(chan *ScheduledMessage[any]),
			dispatchChannel:    make(chan interface{}),
			stopProcess:        make(chan bool)}, args{
			ctx: func() context.Context {
				ctx, _ := context.WithDeadline(context.Background(), time.Now())
				return ctx
			}(),
			drainImmediately: false}, true,
			func(d *Dispatcher[any]) {
				if d.state != shutdown {
					t.Errorf("Shutdown() unexpect state = %v, want Shutdown", d.state)
				}
			}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Dispatcher[any]{
				state:              tt.fields.state,
				pq:                 tt.fields.pq,
				maxMessages:        tt.fields.maxMessages,
				nextMessage:        tt.fields.nextMessage,
				delayer:            tt.fields.delayer,
				delayerIdleChannel: tt.fields.delayerIdleChannel,
				dispatchChannel:    tt.fields.dispatchChannel,
				ingressChannel:     tt.fields.ingressChannel,
				shutdown:           tt.fields.shutdown,
				stopProcess:        tt.fields.stopProcess,
				mutex:              &sync.Mutex{},
			}
			go func(fd *fakeDelayer[any]) {
				if _, ok := <-fd.availableCalled; !ok {
					t.Error("process() expected close of delayer.available()")
				}
				fd.availableCalled = nil
			}(tt.fields.delayer.(*fakeDelayer[any]))
			if err := d.Shutdown(tt.args.ctx, tt.args.drainImmediately); (err != nil) != tt.wantErr {
				t.Errorf("Shutdown() error = %v, wantErr %v", err, tt.wantErr)
			}
			tt.customAssertion(d)
			if msg, ok := <-d.ingressChannel; ok {
				t.Errorf("Shutdown() unexpected message from ingressChannel = %+v", msg)
			}
			if msg, ok := <-d.stopProcess; ok {
				t.Errorf("Shutdown() unexpected message from stopProcess = %+v", msg)
			}
		})
	}
}

func TestDispatcher_integration_inOrderIngress(t *testing.T) {
	skipCI(t)

	dispatcher, _ := NewDispatcher[int](&DispatcherConfig{
		IngressChannelSize:  3,
		DispatchChannelSize: 3,
		MaxMessages:         3,
		GuaranteeOrder:      true,
	})
	ingest := dispatcher.IngressChannel()
	dispatch := dispatcher.DispatchChannel()

	go dispatcher.Start()
	done := make(chan bool)
	go func() {
		i := 0
		for {
			select {
			case <-done:
				return
			case msg, ok := <-dispatch:
				if ok {
					if msg != i {
						t.Errorf("integration; unexpected value from message = %d, want = %d", msg, i)
						t.FailNow()
					}
					i++
				}
			}
		}
	}()

	ingest <- &ScheduledMessage[int]{
		At:      time.Now().Add(time.Duration(0) * time.Second),
		Message: 0,
	}

	ingest <- &ScheduledMessage[int]{
		At:      time.Now().Add(time.Duration(3) * time.Second),
		Message: 1,
	}

	ingest <- &ScheduledMessage[int]{
		At:      time.Now().Add(time.Duration(5) * time.Second),
		Message: 2,
	}

	// block until started
	for dispatcher.state != processing {
	}

	ctx, _ := context.WithTimeout(context.Background(), 10000*time.Millisecond)
	err := dispatcher.Shutdown(ctx, false)
	if err != nil {
		t.Error("integration; failed to drain dispatch channel")
	}
	close(done)
}

func TestDispatcher_integration_outOfOrderIngress(t *testing.T) {
	skipCI(t)

	dispatcher, _ := NewDispatcher[int](&DispatcherConfig{
		IngressChannelSize:  3,
		DispatchChannelSize: 3,
		MaxMessages:         3,
		GuaranteeOrder:      true,
	})
	ingest := dispatcher.IngressChannel()
	dispatch := dispatcher.DispatchChannel()

	go dispatcher.Start()
	done := make(chan bool)
	go func() {
		i := 0
		for {
			select {
			case <-done:
				return
			case msg, ok := <-dispatch:
				if ok {
					if msg != i {
						t.Errorf("integration; unexpected value from message = %d, want = %d", msg, i)
						t.FailNow()
					}
					i++
				}
			}
		}
	}()

	ingest <- &ScheduledMessage[int]{
		At:      time.Now().Add(time.Duration(2) * time.Second),
		Message: 1,
	}

	ingest <- &ScheduledMessage[int]{
		At:      time.Now().Add(time.Duration(0) * time.Second),
		Message: 0,
	}

	ingest <- &ScheduledMessage[int]{
		At:      time.Now().Add(time.Duration(4) * time.Second),
		Message: 2,
	}

	// block until started
	for dispatcher.state != processing {
	}

	ctx, _ := context.WithTimeout(context.Background(), 5000*time.Millisecond)
	err := dispatcher.Shutdown(ctx, false)
	if err != nil {
		t.Error("integration; failed to drain dispatch channel")
	}
	close(done)
}

func TestDispatcher_integration_sameTimeSameOrder(t *testing.T) {
	skipCI(t)

	dispatcher, _ := NewDispatcher[int](&DispatcherConfig{
		IngressChannelSize:  100,
		DispatchChannelSize: 100,
		MaxMessages:         100,
		GuaranteeOrder:      true,
	})
	ingest := dispatcher.IngressChannel()
	dispatch := dispatcher.DispatchChannel()

	go dispatcher.Start()
	done := make(chan bool)
	go func() {
		i := 0
		for {
			select {
			case <-done:
				return
			case msg, ok := <-dispatch:
				if ok {
					if msg != i {
						t.Errorf("integration; unexpected value from message = %d, want = %d", msg, i)
						t.FailNow()
					}
					i++
				}
			}
		}
	}()

	sameTime := time.Now()
	for i := 0; i < 10; i++ {
		ingest <- &ScheduledMessage[int]{
			At:      sameTime,
			Message: i,
		}
	}

	// block until started
	for dispatcher.state != processing {
	}

	ctx, _ := context.WithTimeout(context.Background(), 5000*time.Millisecond)
	err := dispatcher.Shutdown(ctx, false)
	if err != nil {
		t.Error("integration; failed to drain dispatch channel")
	}
	close(done)
}

func RunDispatchLoadTest(b *testing.B, totalMessages int, sameTime bool, config DispatcherConfig) {
	dispatcher, _ := NewDispatcher[int](&config)
	ingest := dispatcher.IngressChannel()
	dispatch := dispatcher.DispatchChannel()
	go dispatcher.Start()

	// block until started
	for dispatcher.state != processing {
	}

	at := time.Now()
	ingestComplete := make(chan bool)
	go func() {
		for i := 0; i < totalMessages; i++ {
			if !sameTime {
				at = time.Now()
			}
			ingest <- &ScheduledMessage[int]{
				At:      at,
				Message: i,
			}
		}
		close(ingestComplete)
	}()

	dispatchComplete := make(chan bool)
	go func() {
		messagesReceived := 0
		for i := 0; i < totalMessages; i++ {
			_, ok := <-dispatch
			if ok {
				messagesReceived++
				// fmt.Println(fmt.Sprintf("%d", msg.(int)))
			}
		}
		if messagesReceived != totalMessages {
			b.Error("benchmark; invalid messages received from dispatch channel")
		}
		close(dispatchComplete)
	}()

	<-ingestComplete
	<-dispatchComplete

	ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(100*time.Millisecond))
	err := dispatcher.Shutdown(ctx, false)
	if err != nil {
		b.Errorf("benchmark; failed to drain dispatch channel: %v", err)
	}
}

func Benchmark_integration_unordered(b *testing.B) {
	for n := 0; n < b.N; n++ {
		RunDispatchLoadTest(b, 1000, false, DispatcherConfig{
			IngressChannelSize:  1000,
			DispatchChannelSize: 1000,
			MaxMessages:         1000,
			GuaranteeOrder:      false,
		})
	}
}

func Benchmark_integration_unorderedSmallBuffer(b *testing.B) {
	for n := 0; n < b.N; n++ {
		RunDispatchLoadTest(b, 1000, false, DispatcherConfig{
			IngressChannelSize:  1,
			DispatchChannelSize: 1,
			MaxMessages:         1000,
			GuaranteeOrder:      false,
		})
	}
}

func Benchmark_integration_unorderedSmallHeap(b *testing.B) {
	for n := 0; n < b.N; n++ {
		RunDispatchLoadTest(b, 1000, false, DispatcherConfig{
			IngressChannelSize:  1000,
			DispatchChannelSize: 1000,
			MaxMessages:         10,
			GuaranteeOrder:      false,
		})
	}
}

func Benchmark_integration_ordered(b *testing.B) {
	for n := 0; n < b.N; n++ {
		RunDispatchLoadTest(b, 1000, false, DispatcherConfig{
			IngressChannelSize:  1000,
			DispatchChannelSize: 1000,
			MaxMessages:         1000,
			GuaranteeOrder:      true,
		})
	}
}

func Benchmark_integration_orderedSmallBuffer(b *testing.B) {
	for n := 0; n < b.N; n++ {
		RunDispatchLoadTest(b, 1000, false, DispatcherConfig{
			IngressChannelSize:  1,
			DispatchChannelSize: 1,
			MaxMessages:         1000,
			GuaranteeOrder:      true,
		})
	}
}

func Benchmark_integration_orderedSmallHeap(b *testing.B) {
	for n := 0; n < b.N; n++ {
		RunDispatchLoadTest(b, 1000, false, DispatcherConfig{
			IngressChannelSize:  1000,
			DispatchChannelSize: 1000,
			MaxMessages:         10,
			GuaranteeOrder:      true,
		})
	}
}

func Benchmark_integration_orderedSameTime(b *testing.B) {
	for n := 0; n < b.N; n++ {
		RunDispatchLoadTest(b, 1000, true, DispatcherConfig{
			IngressChannelSize:  1000,
			DispatchChannelSize: 1000,
			MaxMessages:         1000,
			GuaranteeOrder:      true,
		})
	}
}
