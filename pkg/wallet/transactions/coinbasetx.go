package transactions

import (
	"encoding/binary"
	"errors"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/key"

	"github.com/bwesterb/go-ristretto"
)

type CoinbaseTx struct {
	index     uint32
	netPrefix byte
	r         ristretto.Scalar
	R         ristretto.Point
	Rewards   []*Output
}

func NewCoinBaseTx(netPrefix byte) *CoinbaseTx {

	tx := &CoinbaseTx{}

	// Index for subaddresses
	tx.index = 0

	// prefix to signify testnet/mainnet
	tx.netPrefix = netPrefix

	// randomly generated nonce - r
	var r ristretto.Scalar
	r.Rand()
	tx.setTxPubKey(r)

	return tx
}

func (c *CoinbaseTx) AddReward(pubAddr key.PublicAddress, amount ristretto.Scalar) error {

	if len(c.Rewards)+1 > maxOutputs {
		return errors.New("maximum amount of outputs reached")
	}

	pubKey, err := pubAddr.ToKey(c.netPrefix)
	if err != nil {
		return err
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

func (c *CoinbaseTx) Encode(w io.Writer) error {

	err := binary.Write(w, binary.BigEndian, c.R.Bytes())
	if err != nil {
		return err
	}

	lenRewards := uint32(len(c.Rewards))
	err = binary.Write(w, binary.BigEndian, lenRewards)
	if err != nil {
		return err
	}
	for _, output := range c.Rewards {
		err = output.Encode(w)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *CoinbaseTx) Decode(r io.Reader) error {

	var RBytes [32]byte
	err := binary.Read(r, binary.BigEndian, &RBytes)
	if err != nil {
		return err
	}
	c.R.SetBytes(&RBytes)

	var lenRewards uint32
	err = binary.Read(r, binary.BigEndian, &lenRewards)
	if err != nil {
		return err
	}

	c.Rewards = make([]*Output, lenRewards)

	for i := uint32(0); i < lenRewards; i++ {
		output := &Output{}
		err = output.Decode(r)
		if err != nil {
			return err
		}
		c.Rewards[i] = output
	}

	return nil
}

func (s *CoinbaseTx) setTxPubKey(r ristretto.Scalar) {
	s.r = r
	s.R.ScalarMultBase(&r)
}
