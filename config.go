package gopd

type DispatcherConfig struct {
	IngressChannelSize  int
	DispatchChannelSize int
	MaxMessages         int
	GuaranteeOrder      bool
}
