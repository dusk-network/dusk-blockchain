package message

import "strings"

type Certificate struct {
	Ag   Agreement
	Keys [][]byte
}

func NewCertificate(ag Agreement, keys [][]byte) Certificate {
	return Certificate{
		Ag:   ag,
		Keys: keys,
	}
}

func (c Certificate) String() string {
	var sb strings.Builder
	_, _ = sb.WriteString(c.Ag.String())
	// TODO: write committee keys
	return sb.String()
}
