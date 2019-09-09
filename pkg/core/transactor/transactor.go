package transactor

import (
	"bytes"
	"fmt"
	"math/big"

	ristretto "github.com/bwesterb/go-ristretto"
	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	"github.com/dusk-network/dusk-blockchain/pkg/wallet"
	"github.com/dusk-network/dusk-blockchain/pkg/wallet/transactions"
	"github.com/dusk-network/dusk-wallet/key"
)

// TODO: rename
type Transactor struct {
	w  *wallet.Wallet
	db database.DB
}

// Instantiate a new Transactor struct.
func New(w *wallet.Wallet, db database.DB) *Transactor {
	if db == nil {
		_, db = heavy.CreateDBConnection()
	}

	return &Transactor{
		w:  w,
		db: db,
	}
}

func (t *Transactor) CreateStandardTx(amount uint64, address string) (transactions.Transaction, error) {
	if err := t.syncWallet(); err != nil {
		return nil, err
	}

	// Create a new standard tx
	// TODO: customizable fee
	tx, err := t.w.NewStandardTx(cfg.MinFee)
	if err != nil {
		return nil, err
	}

	// Turn amount into a scalar
	amountScalar := ristretto.Scalar{}
	amountScalar.SetBigInt(big.NewInt(0).SetUint64(amount))

	// Send amount to address
	tx.AddOutput(key.PublicAddress(address), amountScalar)

	// Sign tx
	err = t.w.Sign(tx)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func (t *Transactor) CreateStakeTx(amount, lockTime uint64) (transactions.Transaction, error) {
	if err := t.syncWallet(); err != nil {
		return nil, err
	}

	// Turn amount into a scalar
	amountScalar := ristretto.Scalar{}
	amountScalar.SetBigInt(big.NewInt(0).SetUint64(amount))

	// Create a new stake tx
	tx, err := t.w.NewStakeTx(cfg.MinFee, lockTime, amountScalar)
	if err != nil {
		return nil, err
	}

	// Sign tx
	err = t.w.Sign(tx)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func (t *Transactor) CreateBidTx(amount, lockTime uint64) (transactions.Transaction, error) {
	if err := t.syncWallet(); err != nil {
		return nil, err
	}

	// Turn amount into a scalar
	amountScalar := ristretto.Scalar{}
	amountScalar.SetBigInt(big.NewInt(0).SetUint64(amount))

	// Create a new bid tx
	tx, err := t.w.NewBidTx(cfg.MinFee, lockTime, amountScalar)
	if err != nil {
		return nil, err
	}

	// Sign tx
	err = t.w.Sign(tx)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func (t *Transactor) syncWallet() error {
	var totalSpent, totalReceived uint64
	// keep looping until tipHash = currentBlockHash
	for {
		// Get Wallet height
		walletHeight, err := t.w.GetSavedHeight()
		if err != nil {
			t.w.UpdateWalletHeight(0)
		}

		// Get next block using walletHeight and tipHash of the node
		blk, tipHash, err := fetchBlockHeightAndState(t.db, walletHeight)
		if err == database.ErrBlockNotFound {
			break
		}
		if err != nil {
			return fmt.Errorf("error fetching block from node db: %v\n", err)
		}

		// call wallet.CheckBlock
		spentCount, receivedCount, err := t.w.CheckWireBlock(*blk)
		if err != nil {
			return fmt.Errorf("error checking block: %v\n", err)
		}

		totalSpent += spentCount
		totalReceived += receivedCount

		// check if state is equal to the block that we fetched
		if bytes.Equal(tipHash, blk.Header.Hash) {
			break
		}
	}

	return nil
}

func fetchBlockHeightAndState(db database.DB, height uint64) (*block.Block, []byte, error) {
	var blk *block.Block
	var state *database.State
	err := db.View(func(t database.Transaction) error {
		hash, err := t.FetchBlockHashByHeight(height)
		if err != nil {
			return err
		}
		state, err = t.FetchState()
		if err != nil {
			return err
		}

		blk, err = t.FetchBlock(hash)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return blk, state.TipHash, nil
}

func (t *Transactor) Balance() (float64, error) {
	if err := t.syncWallet(); err != nil {
		return 0.0, err
	}

	balance, err := t.w.Balance()
	if err != nil {
		return 0.0, err
	}

	return balance, nil
}
