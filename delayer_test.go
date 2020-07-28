package gosd

import (
	"testing"
	"time"
)

func Test_delay_stop(t *testing.T) {
	type fields struct {
		state         delayState
		egressChannel chan<- interface{}
		cancelChannel chan bool
	}
	tests := []struct {
		name         string
		fields       fields
		cancelLength int
	}{
		{"waiting", fields{state: Waiting, cancelChannel: make(chan bool, 1)}, 1},
		{"idle", fields{state: Idle, cancelChannel: make(chan bool, 1)}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &delay{
				state:         tt.fields.state,
				egressChannel: tt.fields.egressChannel,
				cancelChannel: tt.fields.cancelChannel,
			}
			d.stop(false)
			if len(d.cancelChannel) != tt.cancelLength {
				t.Errorf("stop() unexpected cancel channel length = %d, want %d", len(d.cancelChannel), tt.cancelLength)
			}
		})
	}
}

func Test_delay_wait(t *testing.T) {
	type fields struct {
		state            delayState
		respondIdleState bool
		idleChannel      chan bool
		egressChannel    chan interface{}
		cancelChannel    chan bool
	}
	type args struct {
		msg *ScheduledMessage
	}
	tests := []struct {
		name            string
		fields          fields
		args            args
		customAssertion func(fields, *delay)
	}{
		{"egressMessage", fields{
			respondIdleState: false,
			egressChannel:    make(chan interface{}),
			idleChannel:      make(chan bool)}, args{msg: &ScheduledMessage{At: time.Now()}}, func(f fields, d *delay) {
			if d.state != Waiting {
				t.Errorf("wait() unexpected state = %+v, want Waiting", d.state)
			}
			if _, ok := <-f.egressChannel; !ok {
				t.Errorf("wait() egress channel closed unexpected")
			}
		}},
		{"egressMessageRespondIdleState", fields{
			respondIdleState: true,
			egressChannel:    make(chan interface{}),
			idleChannel:      make(chan bool)}, args{msg: &ScheduledMessage{At: time.Now()}}, func(f fields, d *delay) {
			if d.state != Waiting {
				t.Errorf("wait() unexpected state = %+v, want Waiting", d.state)
			}
			if _, ok := <-f.egressChannel; !ok {
				t.Errorf("wait() egress channel closed unexpected")
			}
			if _, ok := <-f.idleChannel; !ok {
				t.Errorf("wait() egress channel closed unexpected")
			}
			if d.state != Idle {
				t.Errorf("wait() unexpected state = %+v, want Idle", d.state)
			}
		}},
		{"cancelMessage", fields{
			respondIdleState: false,
			cancelChannel:    make(chan bool, 1),
			idleChannel:      make(chan bool)}, args{msg: &ScheduledMessage{At: time.Now()}}, func(f fields, d *delay) {
			if d.state != Waiting {
				t.Errorf("wait() unexpected state = %+v, want Idle", d.state)
			}
			d.cancelChannel <- true
		}},
		{"cancelMessageRespondIdleState", fields{
			respondIdleState: true,
			cancelChannel:    make(chan bool, 1),
			idleChannel:      make(chan bool)}, args{msg: &ScheduledMessage{At: time.Now().Add(10 + time.Second)}}, func(f fields, d *delay) {
			if d.state != Waiting {
				t.Errorf("wait() unexpected state = %+v, want Waiting", d.state)
			}
			d.cancelChannel <- false
			if _, ok := <-f.idleChannel; !ok {
				t.Errorf("wait() egress channel closed unexpected")
			}
			if d.state != Idle {
				t.Errorf("wait() unexpected state = %+v, want Idle", d.state)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &delay{
				state:            tt.fields.state,
				respondIdleState: tt.fields.respondIdleState,
				idleChannel:      tt.fields.idleChannel,
				egressChannel:    tt.fields.egressChannel,
				cancelChannel:    tt.fields.cancelChannel,
			}
			d.wait(tt.args.msg)
			tt.customAssertion(tt.fields, d)
		})
	}
}

func Test_delay_available(t *testing.T) {
	type fields struct {
		state         delayState
		egressChannel chan<- interface{}
		cancel        chan bool
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{"availableIdle", fields{state: Idle}, true},
		{"availableIdle", fields{state: Waiting}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &delay{
				state:         tt.fields.state,
				egressChannel: tt.fields.egressChannel,
				cancelChannel: tt.fields.cancel,
			}
			if got := d.available(); got != tt.want {
				t.Errorf("available() = %v, want %v", got, tt.want)
			}
		})
	}
}
