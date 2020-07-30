# gosd
[![Build Status](https://travis-ci.com/alexsniffin/gosd.svg?branch=master)](https://travis-ci.com/alexsniffin/gosd)
[![Go Report Card](https://goreportcard.com/badge/github.com/alexsniffin/gosd)](https://goreportcard.com/report/github.com/alexsniffin/gosd)
[![codecov](https://codecov.io/gh/alexsniffin/gosd/branch/master/graph/badge.svg)](https://codecov.io/gh/alexsniffin/gosd)

go-schedulable-dispatcher (gosd), is a library for scheduling when to dispatch a message to a channel.

## Implementation
The implementation provides an ease-of-use API with both an ingress (ingest) channel and egress (disptach) channel. Messages are ingested and processed into a heap based priority queue for dispatching. At most two separate goroutines are used, one for processing of messages from the ingest channel and heap and the other as a timer. Order is not guaranteed by default, but can be changed through the config. By not guaranteeing order, performance was tested to be around 20x that of ordered. If strict-ordering isn't critical to your application, it's recommended to keep the default setting.

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
Tested on a 2018 MBP with 2.6 GHz 6-Core Intel Core i7 CPU and 16 GB 2400 MHz DDR4 RAM and 1000 messages per iteration.
```
Benchmark_integration_unordered
Benchmark_integration_unordered-12               	       2	 920913645 ns/op
Benchmark_integration_unorderedSmallBuffer
Benchmark_integration_unorderedSmallBuffer-12    	       2	 917214818 ns/op
Benchmark_integration_unorderedSmallHeap
Benchmark_integration_unorderedSmallHeap-12      	       2	 812184732 ns/op
Benchmark_integration_ordered
Benchmark_integration_ordered-12                 	       1	20459472269 ns/op
Benchmark_integration_orderedSmallBuffer
Benchmark_integration_orderedSmallBuffer-12      	       1	20554470619 ns/op
Benchmark_integration_orderedSmallHeap
Benchmark_integration_orderedSmallHeap-12        	       1	20484544203 ns/op
```
