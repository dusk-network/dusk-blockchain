package wallet

import "golang.org/x/crypto/ed25519"

type PubKey struct {
	ed25519.PublicKey
}

// Verify method wraps around the native ed25519 function
func (p *PubKey) Verify(message, sig []byte) bool {
	return ed25519.Verify(p.PublicKey, message, sig)
}
