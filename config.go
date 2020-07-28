package gosd

// DispatcherConfig config for creating an instance of a Dispatcher
type DispatcherConfig struct {
	IngressChannelSize  int
	DispatchChannelSize int
	MaxMessages         int
	GuaranteeOrder      bool
}
