package wire

// ProtocolVersion is the protocol version being used by the node.
const ProtocolVersion uint32 = 10000

// DuskNetwork represents the magic bytes for different Dusk networks.
type DuskNetwork uint32

const (
	// DevNet represents the Dusk Dev network magic bytes
	DevNet DuskNetwork = 0x6369616f
)
