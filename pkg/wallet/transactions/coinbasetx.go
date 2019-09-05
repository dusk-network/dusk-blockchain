package transactions

import (
	"bytes"
	"errors"

	"github.com/dusk-network/dusk-crypto/hash"
	"github.com/dusk-network/dusk-wallet/key"

	"github.com/bwesterb/go-ristretto"
)

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

func (s *Coinbase) SetTxPubKey(r ristretto.Scalar) {
	s.r = r
	s.R.ScalarMultBase(&r)
}

func (s *Coinbase) CalculateHash() ([]byte, error) {
	if len(s.TxID) != 0 {
		return s.TxID, nil
	}

	buf := new(bytes.Buffer)
	if err := MarshalCoinbase(buf, s); err != nil {
		return nil, err
	}

	txid, err := hash.Sha3256(buf.Bytes())
	if err != nil {
		return nil, err
	}

	s.TxID = txid
	return txid, nil
}

func (s *Coinbase) StandardTx() Standard {
	return Standard{
		Outputs: s.Rewards,
		R:       s.R,
	}
}

func (s *Coinbase) Type() TxType {
	return s.TxType
}

func (s *Coinbase) Equals(t Transaction) bool {
	other, ok := t.(*Coinbase)
	if !ok {
		return false
	}

	if !bytes.Equal(s.R.Bytes(), other.R.Bytes()) {
		return false
	}

	if !bytes.Equal(s.Score, other.Score) {
		return false
	}

	if !bytes.Equal(s.Proof, other.Proof) {
		return false
	}

	if !s.Rewards.Equals(other.Rewards) {
		return false
	}

	return true
}
