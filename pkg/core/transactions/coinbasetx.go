package transactions

import (
	"bytes"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// Coinbase transaction is the transaction that the block generator
// will create in order to be rewarded for their efforts.
type Coinbase struct {
	// TxType represents the transaction type
	TxType TxType
	// TxID is the transaction identifier for the current transaction
	TxID    []byte
	R       []byte
	Score   []byte
	Proof   []byte
	Rewards Outputs
}

// NewCoinbase will return a Coinbase transaction
// given the zkproof, ephemeral key and the block generators Address.
func NewCoinbase(proof, score, R []byte) *Coinbase {
	return &Coinbase{
		Proof: proof,
		Score: score,
		R:     R,
	}
}

// AddReward will add an Output to the Coinbase struct Rewards array.
func (c *Coinbase) AddReward(output *Output) {
	output.EncryptedMask = make([]byte, 1)
	c.Rewards = append(c.Rewards, output)
}

// Encode implements the Encoder interface
func (c *Coinbase) Encode(w io.Writer) error {
	if err := encoding.WriteUint8(w, uint8(c.TxType)); err != nil {
		return err
	}

	if err := encoding.Write256(w, c.R); err != nil {
		return err
	}

	if err := encoding.Write256(w, c.Score); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(w, c.Proof); err != nil {
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

	return nil
}

// Decode implements the Decoder interface
func (c *Coinbase) Decode(r io.Reader) error {

	var Type uint8
	if err := encoding.ReadUint8(r, &Type); err != nil {
		return err
	}
	c.TxType = TxType(Type)

	if err := encoding.Read256(r, &c.R); err != nil {
		return err
	}

	if err := encoding.Read256(r, &c.Score); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &c.Proof); err != nil {
		return err
	}

	lRewards, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
	}

	c.Rewards = make(Outputs, lRewards)
	for i := uint64(0); i < lRewards; i++ {
		c.Rewards[i] = &Output{}
		if err := c.Rewards[i].Decode(r); err != nil {
			return err
		}
	}

	return nil
}

// CalculateHash hashes all of the encoded fields in a tx, if this has not been done already.
// The resulting byte array is also it's identifier
// Implements merkletree.Payload interface
func (c *Coinbase) CalculateHash() ([]byte, error) {
	if len(c.TxID) != 0 {
		return c.TxID, nil
	}

	txid, err := hashBytes(c.Encode)
	if err != nil {
		return nil, err
	}
	c.TxID = txid

	return c.TxID, nil
}

// Type returns the transaction type
// Implements the TypeInfo interface
func (c *Coinbase) Type() TxType {
	return CoinbaseType
}

// StandardTX implements the transaction interface
func (c *Coinbase) StandardTX() Standard {
	return Standard{
		Outputs: c.Rewards,
		R:       c.R,
	}
}

// Equals returns true, if two coinbase tx's are equal
func (c *Coinbase) Equals(t Transaction) bool {

	other, ok := t.(*Coinbase)
	if !ok {
		return false
	}

	if !bytes.Equal(c.R, other.R) {
		return false
	}

	if !bytes.Equal(c.Score, other.Score) {
		return false
	}

	if !bytes.Equal(c.Proof, other.Proof) {
		return false
	}

	if !c.Rewards.Equals(other.Rewards) {
		return false
	}

	return true
}
