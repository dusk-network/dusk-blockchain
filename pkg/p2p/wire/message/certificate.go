package message

import "strings"

// Certificate is the event created upon finalizing a block in the Agreement phase.
// It will contain one of the received Agreement messages, to be used for creating
// a certificate later on, and the keys of all provisioners that voted for the
// winning block.
type Certificate struct {
	Ag   Agreement
	Keys [][]byte
}

// NewCertificate creates a Certificate message with the given fields.
func NewCertificate(ag Agreement, keys [][]byte) Certificate {
	return Certificate{
		Ag:   ag,
		Keys: keys,
	}
}

// String returns a string representation of a Certificate message.
func (c Certificate) String() string {
	var sb strings.Builder
	_, _ = sb.WriteString(c.Ag.String())
	// TODO: write committee keys
	return sb.String()
}
