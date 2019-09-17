package request

import (
	"encoding/binary"
)

// Seeders will provide the list of nodes that will act as seeder
type Seeders struct {
	o Operation
	p []byte
}

// NewSeeders acts as a constructor for Seeders
func NewSeeders(pubKey *[]byte, port *uint16, proof *[]byte) (*Seeders, error) {
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

	return &Seeders{o: RequestSeedersOperation, p: p}, nil
}

// Operation returns the underlying operation for the request
func (r *Seeders) Operation() *Operation {
	return &r.o
}

// Payload returns the contents of the request
func (r *Seeders) Payload() *[]byte {
	return &r.p
}
