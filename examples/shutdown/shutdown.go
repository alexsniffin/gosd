package main

import (
	"context"
	"fmt"
	"time"

	"github.com/alexsniffin/gosd/v2"
)

func main() {
	// create instance of dispatcher
	dispatcher, err := gosd.NewDispatcher[int](&gosd.DispatcherConfig{
		IngressChannelSize:  100,
		DispatchChannelSize: 100,
		MaxMessages:         100,
		GuaranteeOrder:      false,
	})
	if err != nil {
		panic(err)
	}

	// spawn process
	go dispatcher.Start()

	// schedule some messages
	for i := 0; i < 10; i++ {
		dispatcher.IngressChannel() <- &gosd.ScheduledMessage[int]{
			At:      time.Now().Add(time.Duration(i+1) * time.Second),
			Message: i,
		}
	}

	go func() {
		// wait for the message
		for i := 0; i < 9; i++ {
			msg := <-dispatcher.DispatchChannel()

			fmt.Println(msg)
		}
	}()

	// shutdown with a deadline and fail to drain all the messages
	ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
	err = dispatcher.Shutdown(ctx, false)
	if err != nil {
		panic(err)
	}
}
