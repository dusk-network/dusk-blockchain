package sam3

import (
	"encoding/base32"
	"encoding/base64"
	"errors"
)

// I2P encoding specifications
var (
	I2PBase64 = base64.NewEncoding("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-~")
	I2PBase32 = base32.NewEncoding("abcdefghijklmnopqrstuvwxyz234567")
)

// I2PKeys holds the base-64 encoded destination and private key of an I2P address.
type I2PKeys struct {
	Addr string // Base-64 encoding of the I2P destination (the public I2P address).
	Priv string // Base-64 encoding of the destination, the private key, and the signing private key.
}

// IsI2PAddr checks if addr is in the correct format.
func IsI2PAddr(addr string) error {
	if len(addr) > 4096 || len(addr) < 516 {
		return errors.New("not an I2P address")
	}

	buf := make([]byte, I2PBase64.DecodedLen(len(addr)))
	if _, err := I2PBase64.Decode(buf, []byte(addr)); err != nil {
		return errors.New("address is not base64-encoded")
	}

	return nil
}

// AddrFromBytes creates a new I2P address from a byte array.
func AddrFromBytes(addr []byte) (string, error) {
	if len(addr) > 4096 || len(addr) < 387 {
		return "", errors.New("not an I2P address")
	}

	buf := make([]byte, I2PBase64.EncodedLen(len(addr)))
	I2PBase64.Encode(buf, addr)
	return string(buf), nil
}

// AddrToBytes turns an I2P base-64 encoded string into a byte array.
func AddrToBytes(addr string) ([]byte, error) {
	buf := make([]byte, I2PBase64.DecodedLen(len([]byte(addr))))
	if _, err := I2PBase64.Decode(buf, []byte(addr)); err != nil {
		return buf, errors.New("address is not base64-encoded")
	}

	return buf, nil
}

// Base32 returns the base-32 representation of an I2P address.
func Base32(addr string) (string, error) {
	b, err := AddrToBytes(addr)
	if err != nil {
		return "", err
	}

	b32addr := make([]byte, I2PBase32.EncodedLen(len(b)))
	I2PBase32.Encode(b32addr, b)
	return string(b32addr) + ".b32.i2p", nil
}
