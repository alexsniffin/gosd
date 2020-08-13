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
		GuaranteeOrder:      true,
	})
	if err != nil {
		panic(err)
	}

	// spawn process
	go dispatcher.Start()

	// schedule some messages
	for i := 0; i < 10; i++ {
		dispatcher.IngressChannel() <- &gosd.ScheduledMessage{
			At:      time.Now().Add(time.Duration(i+1) * time.Second),
			Message: i,
		}
	}

	// dispatch a single message
	msg := <-dispatcher.DispatchChannel()
	msgStr := msg.(int)
	fmt.Println(msgStr)

	fmt.Println("pausing...")
	// pause the process
	err = dispatcher.Pause()
	if err != nil {
		panic(err)
	}

	// sleep for a couple seconds
	time.Sleep(2 * time.Second)

	fmt.Println("resuming...")
	// resume the process
	go func() {
		err = dispatcher.Resume()
		if err != nil {
			panic(err)
		}
	}()

	// wait for the messages
	for i := 0; i < 9; i++ {
		msg := <-dispatcher.DispatchChannel()

		// type assert
		msgStr := msg.(int)
		fmt.Println(msgStr)
	}

	// shutdown without deadline
	dispatcher.Shutdown(context.Background(), false)
}
