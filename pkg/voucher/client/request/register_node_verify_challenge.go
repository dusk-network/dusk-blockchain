package request

import (
	"encoding/binary"
)

// RegisterNodeVerifyChallenge will register the current node with a remote voucher and make it available as seeder
type RegisterNodeVerifyChallenge struct {
	o Operation
	p []byte
}

// NewRegisterNodeVerifyChallenge acts as a constructor for RegisterNodeVerifyChallenge
func NewRegisterNodeVerifyChallenge(pubKey *[]byte, port *uint16, proof *[]byte) (*RegisterNodeVerifyChallenge, error) {
	p := make([]byte, 2)
	binary.LittleEndian.PutUint16(p, uint16(len(*pubKey)))
	p = append(p, *pubKey...)

	portLe := make([]byte, 2)
	binary.LittleEndian.PutUint16(portLe, *port)
	p = append(p, portLe...)

	proofLe := make([]byte, 2)
	binary.LittleEndian.PutUint16(proofLe, uint16(len(*proof)))
	p = append(p, proofLe...)
	p = append(p, *proof...)

	return &RegisterNodeVerifyChallenge{o: RegisterNodeVerifyChallengeOperation, p: p}, nil
}

// Operation returns the underlying operation for the request
func (r *RegisterNodeVerifyChallenge) Operation() *Operation {
	return &r.o
}

// Payload returns the contents of the request
func (r *RegisterNodeVerifyChallenge) Payload() *[]byte {
	return &r.p
}
