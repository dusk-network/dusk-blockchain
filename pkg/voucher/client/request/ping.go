package request

// Ping will perform a simple ping command on the voucher
type Ping struct {
	o Operation
	p []byte
}

// NewPing acts as a constructor for Ping
func NewPing() (*Ping, error) {
	return &Ping{o: PingOperation, p: []byte{0x00, 0x00}}, nil
}

// Operation returns the underlying operation for the request
func (r *Ping) Operation() *Operation {
	return &r.o
}

// Payload returns the contents of the request
func (r *Ping) Payload() *[]byte {
	return &r.p
}
