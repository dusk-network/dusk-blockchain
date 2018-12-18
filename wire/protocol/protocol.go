package protocol

// DuskNetwork represents the magic bytes for different Dusk networks.
type DuskNetwork uint32

// ServiceFlag indicates the services provided by the Node.
type ServiceFlag uint64

const (
	// ProtocolVersion is the current protocol version
	ProtocolVersion uint32 = 10000

	// DevNet represents the Dusk Dev network magic bytes
	DevNet DuskNetwork = 0x6369616f

	// ServiceFlag indicates the Node's type
	NodePeerService ServiceFlag = 1
	// BloomFilerService ServiceFlag = 2 // Not implemented
	// PrunedNode        ServiceFlag = 3 // Not implemented
	// LightNode         ServiceFlag = 4 // Not implemented
)
