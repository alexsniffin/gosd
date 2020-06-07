package gopd

import (
	"fmt"
	"testing"
	"time"
)

func TestDispatcher_Integration(t *testing.T) {
	dispatcher := NewDispatcher(DispatcherConfig{
		Parallelization:  5,
		IngestBufferSize: 1,
		EgressBufferSize: 10,
		MaxMessages:      10,
	})
	ingest := dispatcher.IngestChannel()
	dispatch := dispatcher.DispatchChannel()

	dispatcher.Start()

	go func() {
		for {
			select {
			case msg := <-dispatch:
				fmt.Println(msg)
			}
		}
	}()

	for i := 0; i < 100; i++ {
		timeAt := time.Duration(i*2) * time.Second
		ingest <- ScheduledMessage{
			At:      time.Now().Add(timeAt),
			Message: i,
		}
	}

	time.Sleep(150 * time.Second)
}
