package gopd

type DispatcherConfig struct {
	Parallelization   int
	SleepWorkersCount int
	IngressBufferSize int
	EgressBufferSize  int
	MaxMessages       int
}
