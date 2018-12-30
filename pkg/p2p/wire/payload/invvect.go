package payload

// InvType defines the inventory types known in the protocol.
type InvType uint8

// Inventory vector identifiers
const (
	InvTx    InvType = 0x01
	InvBlock InvType = 0x02
)

// InvVect defines an inventory vector in an inv message.
type InvVect struct {
	Type InvType
	Hash []byte
}
