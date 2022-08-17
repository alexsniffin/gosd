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

	// schedule some messages far into the future
	for i := 0; i < 10; i++ {
		dispatcher.IngressChannel() <- &gosd.ScheduledMessage[int]{
			At:      time.Now().Add(time.Duration(i+1000) * time.Hour),
			Message: i,
		}
	}

	// shutdown with a deadline and drain all of the messages
	ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(50*time.Second))
	err = dispatcher.Shutdown(ctx, true)
	if err != nil {
		panic(err)
	}

	for i := 0; i < 10; i++ {
		msg := <-dispatcher.DispatchChannel()

		fmt.Println(msg)
	}
}
