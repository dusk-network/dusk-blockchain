package crypto

import (
	"bytes"
	"crypto/rand"
	"errors"
	"math/big"
	"strconv"

	"github.com/toghrulmaharramov/dusk-go/crypto/base58"
	"github.com/toghrulmaharramov/dusk-go/crypto/hash"
)

func RandEntropy(n uint32) ([]byte, error) {

	b := make([]byte, n)
	a, err := rand.Read(b)

	if err != nil {
		return nil, errors.New("Error generating entropy " + err.Error())
	}
	if uint32(a) != n {
		return nil, errors.New("Error expected to read" + strconv.Itoa(int(n)) + " bytes instead read " + strconv.Itoa(a) + " bytes")
	}
	return b, nil
}

func KeyToAddress(prefix *big.Int, key []byte, padding int) (string, error) {
	buf := new(bytes.Buffer)

	buf.Write(prefix.Bytes())
	pad := make([]byte, padding)
	buf.Write(pad)
	buf.Write(key)

	checksum, err := Checksum(key)
	if err != nil {
		return "", errors.New("Could not calculate the checksum")
	}
	buf.Write(checksum)

	WIF := base58.Encode(buf.Bytes())
	return WIF, nil
}

func Checksum(data []byte) ([]byte, error) {
	hash, err := hash.DoubleSha3256(data)
	if err != nil {
		return nil, err
	}
	return hash[:4], err
}
