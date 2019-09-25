package transactor

import (
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/wallet"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing/chainsync"
)

var (

	// RPCBus methods handled by Transactor
	createWalletChan   = make(chan wire.Req)
	createFromSeedChan = make(chan wire.Req)
	loadWalletChan     = make(chan wire.Req)
	sendBidTxChan      = make(chan wire.Req)
	sendStakeTxChan    = make(chan wire.Req)
	sendStandardTxChan = make(chan wire.Req)
	getBalanceChan     = make(chan wire.Req)
)

// TODO: rename
type Transactor struct {
	w  *wallet.Wallet
	db database.DB
	eb *wire.EventBus
	rb *wire.RPCBus

	// Passed to the consensus component startup
	c *chainsync.Counter
}

// Instantiate a new Transactor struct.
func New(eb *wire.EventBus, rb *wire.RPCBus, db database.DB, counter *chainsync.Counter) (*Transactor, error) {
	if db == nil {
		_, db = heavy.CreateDBConnection()
	}

	t := &Transactor{
		w:  nil,
		db: db,
		eb: eb,
		rb: rb,
		c: counter,
	}

	err := t.registerMethods()
	return t, err
}

// registers all rpcBus channels
func (t *Transactor) registerMethods() error {

	if err := t.rb.Register(wire.LoadWallet, loadWalletChan); err != nil {
		return err
	}

	if err := t.rb.Register(wire.CreateWallet, createWalletChan); err != nil {
		return err
	}

	if err := t.rb.Register(wire.CreateFromSeed, createFromSeedChan); err != nil {
		return err
	}

	if err := t.rb.Register(wire.SendBidTx, sendBidTxChan); err != nil {
		return err
	}

	if err := t.rb.Register(wire.SendStakeTx, sendStakeTxChan); err != nil {
		return err
	}

	if err := t.rb.Register(wire.SendStandardTx, sendStandardTxChan); err != nil {
		return err
	}

	if err := t.rb.Register(wire.GetBalance, getBalanceChan); err != nil {
		return err
	}

	return nil
}
