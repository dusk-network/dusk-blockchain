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

// Encode a Coinbase struct and write to w.
// Implements TypeInfo interface.
func (c *Coinbase) Encode(w io.Writer) error {
	if err := encoding.WriteVarBytes(w, c.Proof); err != nil {
		return err
	}

	if err := encoding.Write256(w, c.EphemeralKey); err != nil {
		return err
	}

	if err := encoding.WriteVarInt(w, uint64(len(c.Rewards))); err != nil {
		return err
	}

	for _, output := range c.Rewards {
		if err := output.Encode(w); err != nil {
			return err
		}
	}

	if err := encoding.Write256(w, c.GeneratorAddress); err != nil {
		return err
	}

	return nil
}

// Decode a Coinbase struct from r.
// Implements TypeInfo interface.
func (c *Coinbase) Decode(r io.Reader) error {
	if err := encoding.ReadVarBytes(r, &c.Proof); err != nil {
		return err
	}

	if err := encoding.Read256(r, &c.EphemeralKey); err != nil {
		return err
	}

	lRewards, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
	}

	c.Rewards = make([]*Output, lRewards)
	for i := uint64(0); i < lRewards; i++ {
		c.Rewards[i] = &Output{}
		if err := c.Rewards[i].Decode(r); err != nil {
			return err
		}
	}

	if err := encoding.Read256(r, &c.GeneratorAddress); err != nil {
		return err
	}

	return nil
}

// Type returns the associated TxType for the Coinbase struct.
// Implements TypeInfo interface.
func (c *Coinbase) Type() TxType {
	return CoinbaseType
}
