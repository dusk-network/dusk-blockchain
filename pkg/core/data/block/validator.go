package block

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/dusk-network/dusk-crypto/hash"
)

// SHA3Payload is a wrapper to the ContractCallTx which is used to facilitate
// its hashing. It is an implementation of merkletree.Payload
type SHA3Payload struct {
	hash []byte
}

func NewSHA3Payload(tx transactions.ContractCall) (SHA3Payload, error) {
	out, err := transactions.Marshal(tx)
	if err != nil {
		return SHA3Payload{}, err
	}

	h, herr := hash.Sha3256(out.Bytes())
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
