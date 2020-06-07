package gopd

type DispatcherConfig struct {
	Parallelization  int
	IngestBufferSize int
	EgressBufferSize int
	MaxMessages      int
}
