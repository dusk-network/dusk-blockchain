package transactions

import (
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// Coinbase defines the transaction info for a coinbase transaction
type Coinbase struct {
	Proof            []byte
	EphemeralKey     []byte
	Rewards          []*Output
	GeneratorAddress []byte
}

// NewCoinbase will return a Coinbase struct with the passed parameters.
func NewCoinbase(proof, ephemeralKey, generatorAddress []byte) *Coinbase {
	return &Coinbase{
		Proof:            proof,
		EphemeralKey:     ephemeralKey,
		GeneratorAddress: generatorAddress,
	}
}

// AddReward will add an Output to the Coinbase struct Rewards array.
func (c *Coinbase) AddReward(output *Output) {
	c.Rewards = append(c.Rewards, output)
}

// Decode a Coinbase struct from r and return it.
func decodeCoinbaseTransaction(r io.Reader) (*Coinbase, error) {
	var proof []byte
	if err := encoding.ReadVarBytes(r, &proof); err != nil {
		return nil, err
	}

	var ephemeralKey []byte
	if err := encoding.Read256(r, &ephemeralKey); err != nil {
		return nil, err
	}

	rewards, err := decodeOutputs(r)
	if err != nil {
		return nil, err
	}

	var generatorAddress []byte
	if err := encoding.Read256(r, &generatorAddress); err != nil {
		return nil, err
	}

	return &Coinbase{
		Proof:            proof,
		EphemeralKey:     ephemeralKey,
		Rewards:          rewards,
		GeneratorAddress: generatorAddress,
	}, nil
}

// Type returns the associated TxType for the Coinbase struct.
// Implements TypeInfo interface.
func (c *Coinbase) Type() TxType {
	return CoinbaseType
}
