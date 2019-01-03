package base58

import "errors"

// Alphabet is a a b58 alphabet.
type Alphabet struct {
	decode [128]int8
	encode [58]byte
}

// NewAlphabet creates a new alphabet from the passed string.
//
// It returns an error if the passed string is not 58 bytes long or isn't valid ASCII.
func NewAlphabet(s string) (*Alphabet, error) {
	if len(s) != 58 {
		return nil, errors.New("base58 alphabets must be 58 bytes long")
	}
	ret := new(Alphabet)
	copy(ret.encode[:], s)
	for i := range ret.decode {
		ret.decode[i] = -1
	}
	for i, b := range ret.encode {
		ret.decode[b] = int8(i)
	}
	return ret, nil
}

// base alphabet
var ab = ("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")
