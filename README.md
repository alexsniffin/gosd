# gosd
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go)
[![Build Status](https://travis-ci.com/alexsniffin/gosd.svg?branch=master)](https://travis-ci.com/alexsniffin/gosd)
[![Go Report Card](https://goreportcard.com/badge/github.com/alexsniffin/gosd)](https://goreportcard.com/report/github.com/alexsniffin/gosd)
[![codecov](https://codecov.io/gh/alexsniffin/gosd/branch/master/graph/badge.svg)](https://codecov.io/gh/alexsniffin/gosd)

go-schedulable-dispatcher (gosd), is a library for scheduling when to dispatch a message to a channel.

## Implementation
The implementation provides an interactive API to handle scheduling messages with a dispatcher. Messages are ingested and processed into a heap based priority queue. Order is not guaranteed by default when messages have the same scheduled time, but can be changed through the config. By guaranteeing order, performance will be slightly worse. If strict-ordering isn't critical to your application, it's recommended to keep the default setting.

## Example
```go
// create instance of dispatcher
dispatcher, err := gosd.NewDispatcher[string](&gosd.DispatcherConfig{
    IngressChannelSize:  100,
    DispatchChannelSize: 100,
    MaxMessages:         100,
    GuaranteeOrder:      false,
})
checkErr(err)

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
```

More examples under [examples](examples).

## Benchmarking
Tested with Go 1.19 and 1000 messages per iteration.
```
goos: windows
goarch: amd64
pkg: github.com/alexsniffin/gosd/v2
cpu: Intel(R) Core(TM) i7-8700K CPU @ 3.70GHz
Benchmark_integration_unordered
Benchmark_integration_unordered-12                           307           3690528 ns/op
Benchmark_integration_unorderedSmallBuffer
Benchmark_integration_unorderedSmallBuffer-12                274           4120104 ns/op
Benchmark_integration_unorderedSmallHeap
Benchmark_integration_unorderedSmallHeap-12                  348           3452703 ns/op
Benchmark_integration_ordered
Benchmark_integration_ordered-12                             135           8650709 ns/op
Benchmark_integration_orderedSmallBuffer
Benchmark_integration_orderedSmallBuffer-12                  207           5867338 ns/op
Benchmark_integration_orderedSmallHeap
Benchmark_integration_orderedSmallHeap-12                    350           3592990 ns/op
Benchmark_integration_orderedSameTime
Benchmark_integration_orderedSameTime-12                     133           8909311 ns/op
```
