package gopd

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestDispatcher_Integration(t *testing.T) {
	dispatcher := NewDispatcher(DispatcherConfig{
		Parallelization:   1,
		SleepWorkersCount: 1,
		IngressBufferSize: 100000,
		EgressBufferSize:  100000,
		MaxMessages:       10000,
	})
	ingest := dispatcher.IngressChannel()
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

	for i := 0; i < 100000; i++ {
		timeAt := time.Duration(rand.Intn(1)+rand.Intn(3)) * time.Second
		ingest <- ScheduledMessage{
			At:      time.Now().Add(timeAt),
			Message: i,
		}
	}

	time.Sleep(150 * time.Second)
}
