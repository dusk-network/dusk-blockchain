package transactions

import (
	"bytes"
	"encoding/binary"
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/key"
	"github.com/dusk-network/dusk-crypto/hash"

	"github.com/bwesterb/go-ristretto"
)

// Coinbase transaction
type Coinbase struct {
	//// Encoded fields
	TxType
	R       ristretto.Point
	Score   []byte
	Proof   []byte
	Rewards Outputs

	//// Non-encoded fields
	index     uint32
	netPrefix byte
	r         ristretto.Scalar
	TxID      []byte
}

// NewCoinbase creates a new coinbase
func NewCoinbase(proof, score []byte, netPrefix byte) *Coinbase {
	tx := &Coinbase{
		TxType:    CoinbaseType,
		Score:     score,
		Proof:     proof,
		index:     0,
		netPrefix: netPrefix,
	}

	// randomly generated nonce - r
	var r ristretto.Scalar
	r.Rand()
	tx.SetTxPubKey(r)

	return tx
}

// AddReward adds the reward to the coinbase outputs
func (c *Coinbase) AddReward(pubKey key.PublicKey, amount ristretto.Scalar) error {
	if len(c.Rewards)+1 > maxOutputs {
		return errors.New("maximum amount of outputs reached")
	}

	stealthAddr := pubKey.StealthAddress(c.r, c.index)

	output := &Output{
		PubKey:          *stealthAddr,
		EncryptedAmount: amount,
	}

	c.Rewards = append(c.Rewards, output)
	c.index = c.index + 1
	return nil
}

// SetTxPubKey sets the public key of the transaction
func (c *Coinbase) SetTxPubKey(r ristretto.Scalar) {
	c.r = r
	c.R.ScalarMultBase(&r)
}

// CalculateHash calculates the hash of the coinbase tx
func (c *Coinbase) CalculateHash() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := marshalCoinbase(buf, c); err != nil {
		return nil, err
	}

	txid, err := hash.Sha3256(buf.Bytes())
	if err != nil {
		return nil, err
	}

	return txid, nil
}

// StandardTx returns the Standard form of the transaction
func (c *Coinbase) StandardTx() *Standard {
	return &Standard{
		Outputs: c.Rewards,
		R:       c.R,
	}
}

// Type returns the type of the transaction
func (c *Coinbase) Type() TxType {
	return c.TxType
}

// Equals tests if a transaction is equal to this coinbase tx
func (c *Coinbase) Equals(t Transaction) bool {
	other, ok := t.(*Coinbase)
	if !ok {
		return false
	}

	if !bytes.Equal(c.R.Bytes(), other.R.Bytes()) {
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

// LockTime is 0 for a coinbase transaction
func (c *Coinbase) LockTime() uint64 {
	return 0
}

func marshalCoinbase(b *bytes.Buffer, c *Coinbase) error {
	if err := binary.Write(b, binary.LittleEndian, c.TxType); err != nil {
		return err
	}

	if err := binary.Write(b, binary.BigEndian, c.R.Bytes()); err != nil {
		return err
	}

	if err := binary.Write(b, binary.BigEndian, c.Score); err != nil {
		return err
	}

	if err := writeVarInt(b, uint64(len(c.Proof))); err != nil {
		return err
	}

	if err := binary.Write(b, binary.BigEndian, c.Proof); err != nil {
		return err
	}

	for _, output := range c.Rewards {
		if err := marshalOutput(b, output); err != nil {
			return err
		}
	}

	return nil
}
