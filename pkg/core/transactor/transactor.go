package transactor

import (
	"github.com/dusk-network/dusk-blockchain/pkg/core/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing/chainsync"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-blockchain/pkg/wallet"
)

var (

	// RPCBus methods handled by Transactor
	createWalletChan   = make(chan rpcbus.Req)
	createFromSeedChan = make(chan rpcbus.Req)
	loadWalletChan     = make(chan rpcbus.Req)
	sendBidTxChan      = make(chan rpcbus.Req)
	sendStakeTxChan    = make(chan rpcbus.Req)
	sendStandardTxChan = make(chan rpcbus.Req)
	getBalanceChan     = make(chan rpcbus.Req)
)

// TODO: rename
type Transactor struct {
	w  *wallet.Wallet
	db database.DB
	eb eventbus.Broker
	rb *rpcbus.RPCBus

	// Passed to the consensus component startup
	c                 *chainsync.Counter
	acceptedBlockChan chan block.Block
}

// Instantiate a new Transactor struct.
func New(eb eventbus.Broker, rb *rpcbus.RPCBus, db database.DB, counter *chainsync.Counter) (*Transactor, error) {
	if db == nil {
		_, db = heavy.CreateDBConnection()
	}

	t := &Transactor{
		w:  nil,
		db: db,
		eb: eb,
		rb: rb,
		c:  counter,
	}

	err := t.registerMethods()
	if err != nil {
		return nil, err
	}

	// topics.AcceptedBlock will be published by Chain subsystem when new block is accepted into blockchain
	t.acceptedBlockChan, _ = consensus.InitAcceptedBlockUpdate(eb)
	return t, err
}

// registers all rpcBus channels
func (t *Transactor) registerMethods() error {

	if err := t.rb.Register(rpcbus.LoadWallet, loadWalletChan); err != nil {
		return err
	}

	if err := t.rb.Register(rpcbus.CreateWallet, createWalletChan); err != nil {
		return err
	}

	if err := t.rb.Register(rpcbus.CreateFromSeed, createFromSeedChan); err != nil {
		return err
	}

	if err := t.rb.Register(rpcbus.SendBidTx, sendBidTxChan); err != nil {
		return err
	}

	if err := t.rb.Register(rpcbus.SendStakeTx, sendStakeTxChan); err != nil {
		return err
	}

	if err := t.rb.Register(rpcbus.SendStandardTx, sendStandardTxChan); err != nil {
		return err
	}

	if err := t.rb.Register(rpcbus.GetBalance, getBalanceChan); err != nil {
		return err
	}

	return nil
}
