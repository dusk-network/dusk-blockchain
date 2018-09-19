package wallet

import (
	"bytes"
	"errors"
	"math/big"

	"github.com/decred/base58"
	"golang.org/x/crypto/ed25519"
)

// PubKey is a wrapper around the ed25519 public key
type PubKey struct {
	ed25519.PublicKey
}

// Verify method wraps around the native ed25519 function
func (p *PubKey) Verify(message, sig []byte) bool {
	return ed25519.Verify(p.PublicKey, message, sig)
}

// Address returns the Base58 encoding of a public key
// Format will start with DUSK
func (p *PubKey) Address() (string, error) {
	if len(p.PublicKey) != 32 {
		return "", errors.New("Pubkey length does not equal 32")
	}
	return pubKeyToAddress(p.PublicKey)
}

func pubKeyToAddress(pub []byte) (string, error) {
	buf := new(bytes.Buffer)

	prefix := big.NewInt(int64(0xFB07A4))

	buf.Write(prefix.Bytes())
	buf.WriteByte(0x00)
	buf.Write(pub)

	checksum, err := Checksum(pub)
	if err != nil {
		return "", errors.New("Could not calculate the checksum")
	}
	buf.Write(checksum)

	WIF := base58.Encode(buf.Bytes())
	return WIF, nil
}
