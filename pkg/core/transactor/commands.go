package transactor

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"math/big"

	ristretto "github.com/bwesterb/go-ristretto"
	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	walletdb "github.com/dusk-network/dusk-wallet/v2/database"
	"github.com/dusk-network/dusk-wallet/v2/key"
	"github.com/dusk-network/dusk-wallet/v2/transactions"
	"github.com/dusk-network/dusk-wallet/v2/wallet"
)

var testnet = byte(2)

func (t *Transactor) loadWallet(password string) (string, error) {
	// First load the database
	db, err := walletdb.New(cfg.Get().Wallet.Store)
	if err != nil {
		return "", err
	}

	// Then load the wallet
	w, err := wallet.LoadFromFile(testnet, db, t.fetchDecoys, t.fetchInputs, password, cfg.Get().Wallet.File)
	if err != nil {
		db.Close()
		return "", err
	}

	walletAddr, err := w.PublicAddress()
	if err != nil {
		db.Close()
		return "", err
	}

	t.w = w
	return walletAddr, nil
}

func (t *Transactor) createWallet(password string) (string, error) {
	db, err := walletdb.New(cfg.Get().Wallet.Store)
	if err != nil {
		return "", err
	}

	w, err := wallet.New(rand.Read, testnet, db, t.fetchDecoys, t.fetchInputs, password, cfg.Get().Wallet.File)
	if err != nil {
		db.Close()
		return "", err
	}

	walletAddr, err := w.PublicAddress()
	if err != nil {
		db.Close()
		return "", err
	}

	t.w = w
	return walletAddr, nil
}

func (t *Transactor) createFromSeed(seed string, password string) (string, error) {

	seedBytes, err := hex.DecodeString(seed)
	if err != nil {
		return "", fmt.Errorf("error attempting to decode seed: %v\n", err)
	}

	// First load the database
	db, err := walletdb.New(cfg.Get().Wallet.Store)
	if err != nil {
		return "", err
	}

	// Then load the wallet
	w, err := wallet.LoadFromSeed(seedBytes, testnet, db, t.fetchDecoys, t.fetchInputs, password, cfg.Get().Wallet.File)
	if err != nil {
		db.Close()
		return "", err
	}

	walletAddr, err := w.PublicAddress()
	if err != nil {
		db.Close()
		return "", err
	}

	t.w = w
	return walletAddr, nil
}

func (t *Transactor) CreateStandardTx(amount uint64, address string) (transactions.Transaction, error) {

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
	if err := tx.AddOutput(key.PublicAddress(address), amountScalar); err != nil {
		return nil, err
	}

	// Sign tx
	err = t.w.Sign(tx)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func (t *Transactor) CreateStakeTx(amount, lockTime uint64) (transactions.Transaction, error) {

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

	walletAddr, err := t.w.PublicAddress()
	if err != nil {
		return err
	}

	if totalSpent > 0 || totalReceived > 0 {
		log.Infof("Wallet: %s - TotalReceived %d, TotalSpent %d", walletAddr, totalReceived, totalSpent)
	}

	return nil
}

// Balance returns both unlocked wallet balance and locked wallet balance that corresponds to the loaded wallet
func (t *Transactor) Balance() (uint64, uint64, error) {
	// retrieve balance from wallet unspent inputs
	return t.w.Balance()
}

func (t *Transactor) Address() (string, error) {
	return t.w.PublicAddress()
}
