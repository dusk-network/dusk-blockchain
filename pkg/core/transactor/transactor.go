package transactor

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"

	ristretto "github.com/bwesterb/go-ristretto"
	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	"github.com/dusk-network/dusk-blockchain/pkg/wallet"
	walletdb "github.com/dusk-network/dusk-blockchain/pkg/wallet/database"
	"github.com/dusk-network/dusk-blockchain/pkg/wallet/transactions"
	"github.com/dusk-network/dusk-crypto/mlsag"
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

func (t *Transactor) Listen() {
	for {
		select {
		case r := <-wire.CreateWalletChan:
			if t.w != nil {
				r.ErrChan <- errors.New("wallet is already loaded")
				continue
			}

			t.createWallet(string(r.Params.Bytes()))
		case r := <-wire.LoadWalletChan:
			if t.w != nil {
				r.ErrChan <- errors.New("wallet is already loaded")
				continue
			}

			t.loadWallet(string(r.Params.Bytes()))
		case r := <-wire.SendTxChan:
			if t.w == nil {
				r.ErrChan <- errors.New("wallet is not loaded yet")
				continue
			}

			txType := make([]byte, 1)
			if _, err := &r.Params.Read(txType); err != nil {
				r.ErrChan <- err
				continue
			}

			amountBytes := make([]byte, 8)
			if _, err := &r.Params.Read(amountBytes); err != nil {
				r.ErrChan <- err
				continue
			}

			amount := binary.LittleEndian.Uint64(amountBytes)
			var tx transactions.Transaction
			var err error
			switch transactions.TxType(txType[0]) {
			case transactions.StandardType:
				tx, err = t.CreateStandardTx(amount, string(r.Params.Bytes()))
				if err != nil {
					r.ErrChan <- err
					continue
				}
			case transactions.BidType:
				tx, err = t.CreateBidTx(amount, binary.LittleEndian.Uint64(r.Params.Bytes()))
				if err != nil {
					r.ErrChan <- err
					continue
				}
			case transactions.StakeType:
				tx, err = t.CreateStakeTx(amount, binary.LittleEndian.Uint64(r.Params.Bytes()))
				if err != nil {
					r.ErrChan <- err
					continue
				}
			default:
				r.ErrChan <- errors.New("transaction type was not recognized")
				continue
			}

			respBuf := new(bytes.Buffer)
			if err := transactions.Marshal(respBuf, tx); err != nil {
				r.ErrChan <- err
				continue
			}

			r.RespChan <- *respBuf
		case r := <-wire.GetBalanceChan:
			if t.w == nil {
				r.ErrChan <- errors.New("wallet is not loaded yet")
				continue
			}

			balance, err := t.Balance()
			if err != nil {
				r.ErrChan <- err
				continue
			}

			buf := new(bytes.Buffer)
			binary.Write(buf, binary.LittleEndian, balance)
			r.RespChan <- *buf
		}
	}
}

func (t *Transactor) createWallet(password string) error {
	db, err := walletdb.New(cfg.Get().Wallet.Store)
	if err != nil {
		return err
	}

	w, err := wallet.New(rand.Read, testnet, db, fetchDecoys, fetchInputs, password)
	if err != nil {
		return err
	}
	pubAddr, err := w.PublicAddress()
	if err != nil {
		return err
	}

	t.w = w
	return nil
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

func fetchDecoys(numMixins int) []mlsag.PubKeys {
	_, db := heavy.CreateDBConnection()

	var pubKeys []mlsag.PubKeys
	var decoys []ristretto.Point
	db.View(func(t database.Transaction) error {
		decoys = t.FetchDecoys(numMixins)
		return nil
	})

	// Potential panic if the database does not have enough decoys
	for i := 0; i < numMixins; i++ {

		var keyVector mlsag.PubKeys
		keyVector.AddPubKey(decoys[i])

		var secondaryKey ristretto.Point
		secondaryKey.Rand()
		keyVector.AddPubKey(secondaryKey)

		pubKeys = append(pubKeys, keyVector)
	}
	return pubKeys
}

func fetchInputs(netPrefix byte, db *walletdb.DB, totalAmount int64, key *key.Key) ([]*transactions.Input, int64, error) {
	// Fetch all inputs from database that are >= totalAmount
	// returns error if inputs do not add up to total amount
	privSpend, err := key.PrivateSpend()
	if err != nil {
		return nil, 0, err
	}
	return db.FetchInputs(privSpend.Bytes(), totalAmount)
}
