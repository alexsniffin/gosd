package gosd

type DispatcherConfig struct {
	IngressChannelSize  int
	DispatchChannelSize int
	MaxMessages         int
	GuaranteeOrder      bool
}
