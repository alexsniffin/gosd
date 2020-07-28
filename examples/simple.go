package main

import (
	"context"
	"fmt"
	"time"

	"github.com/alexsniffin/gosd"
)

func main() {
	// create instance of dispatcher
	dispatcher, err := gosd.NewDispatcher(&gosd.DispatcherConfig{
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
	dispatcher.IngressChannel() <- &gosd.ScheduledMessage{
		At:      time.Now().Add(1 * time.Second),
		Message: "Hello World in 1 second!",
	}

	// wait for the message
	msg := <-dispatcher.DispatchChannel()

	// type assert
	msgStr := msg.(string)
	fmt.Println(msgStr)
	// Hello World in 1 second!

	// shutdown without deadline
	dispatcher.Shutdown(context.Background(), false)
}
