# gosd
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go)
[![Build Status](https://travis-ci.com/alexsniffin/gosd.svg?branch=master)](https://travis-ci.com/alexsniffin/gosd)
[![Go Report Card](https://goreportcard.com/badge/github.com/alexsniffin/gosd)](https://goreportcard.com/report/github.com/alexsniffin/gosd)
[![codecov](https://codecov.io/gh/alexsniffin/gosd/branch/master/graph/badge.svg)](https://codecov.io/gh/alexsniffin/gosd)

go-schedulable-dispatcher (gosd), is a library for scheduling when to dispatch a message to a channel.

## Implementation
The implementation provides an ease-of-use API with both an ingress (ingest) channel and egress (dispatch) channel. Messages are ingested and processed into a heap based priority queue for dispatching. At most two separate goroutines are used, one for processing of messages from both the ingest channel and heap then the other as a timer. Order is not guaranteed by default when messages have the same scheduled time, but can be changed through the config. By guaranteeing order, performance will be slightly worse. If strict-ordering isn't critical to your application, it's recommended to keep the default setting.

## Example
```go
// create instance of dispatcher
dispatcher, err := gosd.NewDispatcher(&gosd.DispatcherConfig{
    IngressChannelSize:  100,
    DispatchChannelSize: 100,
    MaxMessages:         100,
    GuaranteeOrder:      false,
})
checkErr(err)

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
```

More examples under [examples](examples).

## Benchmarking
Tested with Intel Core i7-8700K CPU @ 3.70GHz, DDR4 RAM and 1000 messages per iteration.
```
Benchmark_integration_unordered-12               	     142	   8654906 ns/op
Benchmark_integration_unorderedSmallBuffer-12    	     147	   9503403 ns/op
Benchmark_integration_unorderedSmallHeap-12      	     122	   8860732 ns/op
Benchmark_integration_ordered-12                 	      96	  13354174 ns/op
Benchmark_integration_orderedSmallBuffer-12      	     121	  10115702 ns/op
Benchmark_integration_orderedSmallHeap-12        	     129	  10441857 ns/op
Benchmark_integration_orderedSameTime-12        	     99	   	  12575961 ns/op
```
