package wallet

import (
	"math/big"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/syndtr/goleveldb/leveldb"
)

// NewStandardTx creates a new standard transaction
func (w *Wallet) NewStandardTx(fee int64) (*transactions.Standard, error) {
	tx, err := transactions.NewStandard(0, w.netPrefix, fee)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

// NewStakeTx creates a new Stake transaction
func (w *Wallet) NewStakeTx(fee int64, lockTime uint64, amount ristretto.Scalar) (*transactions.Stake, error) {
	edPubBytes := w.consensusKeys.EdPubKeyBytes
	blsPubBytes := w.consensusKeys.BLSPubKeyBytes
	tx, err := transactions.NewStake(0, w.netPrefix, fee, lockTime, edPubBytes, blsPubBytes)
	if err != nil {
		return nil, err
	}

	// Send locked stake amount to self
	walletAddr, err := w.keyPair.PublicKey().PublicAddress(w.netPrefix)
	if err != nil {
		return nil, err
	}

	err = tx.AddOutput(*walletAddr, amount)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

// NewBidTx creates a new bid transaction
func (w *Wallet) NewBidTx(fee int64, lockTime uint64, amount ristretto.Scalar) (*transactions.Bid, error) {
	privateSpend, err := w.keyPair.PrivateSpend()
	if err != nil {
		return nil, err
	}

	privateSpend.Bytes()

	// TODO: index is currently set to be zero.
	// To avoid any privacy implications, the wallet should increment
	// the index by how many bidding txs are seen
	mBytes := generateM(privateSpend.Bytes(), 0)
	tx, err := transactions.NewBid(0, w.netPrefix, fee, lockTime, mBytes)
	if err != nil {
		return nil, err
	}

	// Send bid amount to self
	walletAddr, err := w.keyPair.PublicKey().PublicAddress(w.netPrefix)
	if err != nil {
		return nil, err
	}
	err = tx.AddOutput(*walletAddr, amount)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

// AddInputs adds up the total outputs and fee then fetches inputs to consolidate this
func (w *Wallet) AddInputs(tx *transactions.Standard) error {
	totalAmount := tx.Fee.BigInt().Int64() + tx.TotalSent.BigInt().Int64()
	inputs, changeAmount, err := w.fetchInputs(w.netPrefix, w.db, totalAmount, w.keyPair)
	if err != nil {
		return err
	}
	for _, input := range inputs {
		e := tx.AddInput(input)
		if e != nil {
			return e
		}
	}

	changeAddr, err := w.keyPair.PublicKey().PublicAddress(w.netPrefix)
	if err != nil {
		return err
	}

	// Convert int64 to ristretto value
	var x ristretto.Scalar
	x.SetBigInt(big.NewInt(changeAmount))

	return tx.AddOutput(*changeAddr, x)
}

// Sign a transaction
func (w *Wallet) Sign(tx SignableTx) error {
	// Assuming user has added all of the outputs
	standardTx := tx.StandardTx()

	// Fetch Inputs
	err := w.AddInputs(standardTx)
	if err != nil {
		return err
	}

	// Fetch decoys
	err = standardTx.AddDecoys(numMixins, w.fetchDecoys)
	if err != nil {
		return err
	}

	if err := tx.Prove(); err != nil {
		return err
	}

	// Remove inputs from the db, to prevent accidental double-spend attempts
	// when sending transactions quickly after one another.
	for _, input := range tx.StandardTx().Inputs {
		outputKey, err := w.db.GetPubKey(input.KeyImage.Bytes())
		if err == leveldb.ErrNotFound {
			continue
		}
		if err != nil {
			return err
		}

		_ = w.db.RemoveInput(outputKey, input.KeyImage.Bytes())
	}

	return nil
}
