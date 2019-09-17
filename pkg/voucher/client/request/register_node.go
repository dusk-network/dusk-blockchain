package request

import (
	"encoding/binary"
)

// RegisterNode will register the current node with a remote voucher and make it available as seeder
type RegisterNode struct {
	o Operation
	p []byte
}

// NewRegisterNode acts as a constructor for RegisterNode
func NewRegisterNode(pubKey *[]byte, port *uint16) (*RegisterNode, error) {
	p := make([]byte, 2)
	binary.LittleEndian.PutUint16(p, uint16(len(*pubKey)))
	p = append(p, *pubKey...)

	portLe := make([]byte, 2)
	binary.LittleEndian.PutUint16(portLe, *port)
	p = append(p, portLe...)

	return &RegisterNode{o: RegisterNodeOperation, p: p}, nil
}

// Operation returns the underlying operation for the request
func (r *RegisterNode) Operation() *Operation {
	return &r.o
}

// Payload returns the contents of the request
func (r *RegisterNode) Payload() *[]byte {
	return &r.p
}
