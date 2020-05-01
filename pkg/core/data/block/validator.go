package block

import (
	"bytes"

	"github.com/dusk-network/dusk-crypto/hash"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
	"google.golang.org/protobuf/proto"
)

// SHA3Payload is a wrapper to the ContractCallTx which is used to facilitate
// its hashing. It is an implementation of merkletree.Payload
type SHA3Payload struct {
	hash []byte
}

func newSHA3Payload(tx *rusk.ContractCallTx) (SHA3Payload, error) {
	out, err := proto.Marshal(tx)
	if err != nil {
		return SHA3Payload{}, err
	}

	h, herr := hash.Sha3256(out)
	if herr != nil {
		return SHA3Payload{}, err
	}
	return SHA3Payload{h}, nil
}

// CalculateHash returns the hash of the ContractCallTx
func (s SHA3Payload) CalculateHash() ([]byte, error) {
	return s.hash, nil
}

// Equals tests for equality
func (s SHA3Payload) Equals(other interface{}) bool {
	sother, ok := other.(SHA3Payload)
	if !ok {
		return false
	}
	return bytes.Equal(s.hash, sother.hash)
}
