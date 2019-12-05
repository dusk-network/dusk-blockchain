package transactor

import (
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing/chainsync"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-wallet/block"
	"github.com/dusk-network/dusk-wallet/transactions"
	"github.com/dusk-network/dusk-wallet/wallet"
)

// TODO: rename
type Transactor struct {
	w           *wallet.Wallet
	db          database.DB
	eb          *eventbus.EventBus
	rb          *rpcbus.RPCBus
	fetchDecoys transactions.FetchDecoys
	fetchInputs wallet.FetchInputs
	walletOnly  bool

	// Passed to the consensus component startup
	c                 *chainsync.Counter
	acceptedBlockChan <-chan block.Block

	// rpcbus channels
	createWalletChan   chan rpcbus.Request
	createFromSeedChan chan rpcbus.Request
	loadWalletChan     chan rpcbus.Request
	sendBidTxChan      chan rpcbus.Request
	sendStakeTxChan    chan rpcbus.Request
	sendStandardTxChan chan rpcbus.Request
	getBalanceChan     chan rpcbus.Request
	getAddressChan     chan rpcbus.Request
}

// Instantiate a new Transactor struct.
func New(eb *eventbus.EventBus, rb *rpcbus.RPCBus, db database.DB,
	counter *chainsync.Counter, fdecoys transactions.FetchDecoys,
	finputs wallet.FetchInputs, walletOnly bool) (*Transactor, error) {
	if db == nil {
		_, db = heavy.CreateDBConnection()
	}

	t := &Transactor{
		w:           nil,
		db:          db,
		eb:          eb,
		rb:          rb,
		c:           counter,
		fetchDecoys: fdecoys,
		fetchInputs: finputs,
		walletOnly:  walletOnly,

		createWalletChan:   make(chan rpcbus.Request, 1),
		createFromSeedChan: make(chan rpcbus.Request, 1),
		loadWalletChan:     make(chan rpcbus.Request, 1),
		sendBidTxChan:      make(chan rpcbus.Request, 1),
		sendStakeTxChan:    make(chan rpcbus.Request, 1),
		sendStandardTxChan: make(chan rpcbus.Request, 1),
		getBalanceChan:     make(chan rpcbus.Request, 1),
		getAddressChan:     make(chan rpcbus.Request, 1),
	}

	if t.fetchDecoys == nil {
		t.fetchDecoys = fetchDecoys
	}

	if t.fetchInputs == nil {
		t.fetchInputs = fetchInputs
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

	if err := t.rb.Register(rpcbus.LoadWallet, t.loadWalletChan); err != nil {
		return err
	}

	if err := t.rb.Register(rpcbus.CreateWallet, t.createWalletChan); err != nil {
		return err
	}

	if err := t.rb.Register(rpcbus.CreateFromSeed, t.createFromSeedChan); err != nil {
		return err
	}

	if err := t.rb.Register(rpcbus.SendBidTx, t.sendBidTxChan); err != nil {
		return err
	}

	if err := t.rb.Register(rpcbus.SendStakeTx, t.sendStakeTxChan); err != nil {
		return err
	}

	if err := t.rb.Register(rpcbus.SendStandardTx, t.sendStandardTxChan); err != nil {
		return err
	}

	if err := t.rb.Register(rpcbus.GetBalance, t.getBalanceChan); err != nil {
		return err
	}

	if err := t.rb.Register(rpcbus.GetAddress, t.getAddressChan); err != nil {
		return err
	}

	return nil
}

func (t *Transactor) Wallet() (*wallet.Wallet, error) {

	if t.w == nil {
		return nil, errWalletNotLoaded
	}

	return t.w, nil
}
