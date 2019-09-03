package transactor

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"

	ristretto "github.com/bwesterb/go-ristretto"
	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/wallet"
	"github.com/dusk-network/dusk-wallet/key"
	log "github.com/sirupsen/logrus"
)

var l = log.WithField("process", "transactor")

// The transactor is a process which can send transactions upon request from any other component.
// It is instantiated once the user loads in a wallet file.
type transactor struct {
	publisher wire.EventPublisher
	w         *wallet.Wallet

	// Default values for consensus txs
	lockTime uint64
	amount   ristretto.Scalar

	// transaction triggering channels
	transferChan <-chan transferInfo
	bidChan      <-chan *bytes.Buffer
	stakeChan    <-chan *bytes.Buffer

	// channels to update consensus tx values
	lockTimeChan <-chan uint64
	amountChan   <-chan uint64
}

// Instantiate a new transactor struct.
func newTransactor(eventBroker wire.EventBroker, w *wallet.Wallet) *transactor {
	lockTime := cfg.Get().Consensus.DefaultLockTime
	amount := ristretto.Scalar{}
	amount.SetBigInt(big.NewInt(0).SetUint64(cfg.Get().Consensus.DefaultValue))

	bidChan := make(chan *bytes.Buffer, 1)
	eventBroker.Subscribe(string(topics.Bid), bidChan)
	stakeChan := make(chan *bytes.Buffer, 1)
	eventBroker.Subscribe(string(topics.Stake), stakeChan)
	return &transactor{
		publisher:    eventBroker,
		w:            w,
		lockTime:     lockTime,
		amount:       amount,
		transferChan: initTransferCollector(eventBroker),
		bidChan:      bidChan,
		stakeChan:    stakeChan,
	}
}

// Launch a transactor process.
func Launch(eventBroker wire.EventBroker, w *wallet.Wallet) {
	t := newTransactor(eventBroker, w)
	go t.listen()
}

func (t *transactor) listen() {
	for {
		select {
		case info := <-t.transferChan:
			t.transfer(info.amount, info.address)
		case <-t.bidChan:
			t.sendBid()
		case <-t.stakeChan:
			t.sendStake()
		case lockTime := <-t.lockTimeChan:
			t.lockTime = lockTime
		case amount := <-t.amountChan:
			amountScalar := ristretto.Scalar{}
			amountScalar.SetBigInt(big.NewInt(0).SetUint64(amount))
			t.amount = amountScalar
		}
	}
}

func (t *transactor) transfer(amount uint64, address string) {
	if err := t.syncWallet(); err != nil {
		// log
		return
	}

	// Create a new standard tx
	tx, err := t.w.NewStandardTx(cfg.MinFee)
	if err != nil {
		l.WithError(err).Warnln("error creating transaction")
		return
	}

	// Turn amount into a scalar
	amountScalar := ristretto.Scalar{}
	amountScalar.SetBigInt(big.NewInt(0).SetUint64(amount))

	// Send amount to address
	tx.AddOutput(key.PublicAddress(address), amountScalar)

	// Sign tx
	err = t.w.Sign(tx)
	if err != nil {
		l.WithError(err).Warnln("error signing transaction")
		return
	}

	// Convert wallet-tx to wireTx and encode into buffer
	wireTx, err := tx.WireStandardTx()
	if err != nil {
		l.WithError(err).Warnln("error converting transaction")
		return
	}
	buf := new(bytes.Buffer)
	if err := wireTx.Encode(buf); err != nil {
		l.WithError(err).Warnln("error encoding transaction")
		return
	}

	_, err = wireTx.CalculateHash()
	if err != nil {
		l.WithError(err).Warnln("error calculating transaction hash")
		return
	}
	l.WithField("hash", hex.EncodeToString(wireTx.TxID)).Debugln("transaction created")

	t.publisher.Publish(string(topics.Tx), buf)
}

func (t *transactor) sendStake() {
	if err := t.syncWallet(); err != nil {
		// log
		return
	}

	// Create a new stake tx
	tx, err := t.w.NewStakeTx(cfg.MinFee, t.lockTime, t.amount)
	if err != nil {
		fmt.Fprintf(os.Stdout, "error creating tx: %v\n", err)
		return
	}

	// Sign tx
	err = t.w.Sign(tx)
	if err != nil {
		fmt.Fprintf(os.Stdout, "%s\n", err.Error())
		return
	}

	// Convert wallet-tx to wireTx and encode into buffer
	wireTx, err := tx.WireStakeTx()
	if err != nil {
		fmt.Fprintf(os.Stdout, "%s\n", err.Error())
		return
	}
	buf := new(bytes.Buffer)
	if err := wireTx.Encode(buf); err != nil {
		fmt.Fprintf(os.Stdout, "error encoding tx: %v\n", err)
		return
	}

	_, err = wireTx.CalculateHash()
	if err != nil {
		fmt.Fprintf(os.Stdout, "%s\n", err.Error())
		return
	}
	fmt.Fprintf(os.Stdout, "hash: %s\n", hex.EncodeToString(wireTx.TxID))

	t.publisher.Publish(string(topics.Tx), buf)
}

func (t *transactor) sendBid() {
	if err := t.syncWallet(); err != nil {
		// log
		return
	}

	// Create a new bid tx
	tx, err := t.w.NewBidTx(cfg.MinFee, t.lockTime, t.amount)
	if err != nil {
		fmt.Fprintf(os.Stdout, "error creating tx: %v\n", err)
		return
	}

	// Sign tx
	err = t.w.Sign(tx)
	if err != nil {
		fmt.Fprintf(os.Stdout, "%s\n", err.Error())
		return
	}

	// Convert wallet-tx to wireTx and encode into buffer
	wireTx, err := tx.WireBid()
	if err != nil {
		fmt.Fprintf(os.Stdout, "%s\n", err.Error())
		return
	}
	buf := new(bytes.Buffer)
	if err := wireTx.Encode(buf); err != nil {
		fmt.Fprintf(os.Stdout, "error encoding tx: %v\n", err)
		return
	}

	_, err = wireTx.CalculateHash()
	if err != nil {
		fmt.Fprintf(os.Stdout, "%s\n", err.Error())
		return
	}
	fmt.Fprintf(os.Stdout, "hash: %s\n", hex.EncodeToString(wireTx.TxID))

	t.publisher.Publish(string(topics.Tx), buf)
}

func (t *transactor) syncWallet() error {
	var totalSpent, totalReceived uint64
	_, db := heavy.CreateDBConnection()
	// keep looping until tipHash = currentBlockHash
	for {
		// Get Wallet height
		walletHeight, err := t.w.GetSavedHeight()
		if err != nil {
			t.w.UpdateWalletHeight(0)
		}

		// Get next block using walletHeight and tipHash of the node
		blk, tipHash, err := fetchBlockHeightAndState(db, walletHeight)
		if err != nil {
			return fmt.Errorf("error fetching block from node db: %v\n", err)
		}

		// call wallet.CheckBlock
		spentCount, receivedCount, err := t.w.CheckWireBlock(*blk)
		if err != nil {
			return fmt.Errorf("error fetching block: %v\n", err)
		}

		totalSpent += spentCount
		totalReceived += receivedCount

		// check if state is equal to the block that we fetched
		if bytes.Equal(tipHash, blk.Header.Hash) {
			break
		}
	}

	l.WithFields(log.Fields{
		"spends":   totalSpent,
		"receives": totalReceived,
	}).Debugln("finished wallet sync")
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

func (t *transactor) balance(args []string, publisher wire.EventBroker, rpcBus *wire.RPCBus) float64 {
	if err := t.syncWallet(); err != nil {
		l.WithError(err).Warnln("error syncing wallet")
		return 0.0
	}

	balance, err := t.w.Balance()
	if err != nil {
		l.WithError(err).Warnln("error fetching balance")
		return 0.0
	}

	return balance
}
