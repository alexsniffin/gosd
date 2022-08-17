package main

import (
	"context"
	"fmt"
	"time"

	"github.com/alexsniffin/gosd/v2"
)

func main() {
	// create instance of dispatcher
	dispatcher, err := gosd.NewDispatcher[string](&gosd.DispatcherConfig{
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

	// schedule a message
	dispatcher.IngressChannel() <- &gosd.ScheduledMessage[string]{
		At:      time.Now().Add(1 * time.Second),
		Message: "Hello World in 1 second!",
	}

	// wait for the message
	msg := <-dispatcher.DispatchChannel()

	fmt.Println(msg)
	// Hello World in 1 second!

	// shutdown without deadline
	dispatcher.Shutdown(context.Background(), false)
}
