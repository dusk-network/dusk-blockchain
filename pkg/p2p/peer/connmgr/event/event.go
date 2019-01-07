package event

// MonEventType defines the monitoring event types
type MonEventType uint8

// Mnitoring events
const (
	MinPeersExcd        MonEventType = 0x01
	MaxPeersExcd        MonEventType = 0x02
	OutOfPeerAddr       MonEventType = 0x03
	MinGetAddrPeersExcd MonEventType = 0x04
)

// MonEvent defines an monitor event
type MonEvent struct {
	Type    MonEventType
	Enabled bool
}
