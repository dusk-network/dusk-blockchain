package transactor

import (
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/maintainer"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/wallet"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing/chainsync"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	zkproof "github.com/dusk-network/dusk-zkproof"
)

//nolint
type Transactor struct { // TODO: rename
	w                 *wallet.Wallet
	db                database.DB
	eb                *eventbus.EventBus
	rb                *rpcbus.RPCBus
	fetchDecoys       transactions.FetchDecoys
	fetchInputs       wallet.FetchInputs
	walletOnly        bool
	maintainerStarted bool

	// Passed to the consensus component startup
	c                 *chainsync.Counter
	acceptedBlockChan <-chan block.Block

	// rpcbus channels
	createWalletChan          chan rpcbus.Request
	createFromSeedChan        chan rpcbus.Request
	loadWalletChan            chan rpcbus.Request
	sendBidTxChan             chan rpcbus.Request
	sendStakeTxChan           chan rpcbus.Request
	sendStandardTxChan        chan rpcbus.Request
	getBalanceChan            chan rpcbus.Request
	getUnconfirmedBalanceChan chan rpcbus.Request
	getAddressChan            chan rpcbus.Request
	getTxHistoryChan          chan rpcbus.Request
	automateConsensusTxsChan  chan rpcbus.Request
	isWalletLoadedChan        chan rpcbus.Request
	clearWalletDatabaseChan   chan rpcbus.Request
}

// New Instantiate a new Transactor struct.
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

		createWalletChan:          make(chan rpcbus.Request, 1),
		createFromSeedChan:        make(chan rpcbus.Request, 1),
		loadWalletChan:            make(chan rpcbus.Request, 1),
		sendBidTxChan:             make(chan rpcbus.Request, 1),
		sendStakeTxChan:           make(chan rpcbus.Request, 1),
		sendStandardTxChan:        make(chan rpcbus.Request, 1),
		getBalanceChan:            make(chan rpcbus.Request, 1),
		getUnconfirmedBalanceChan: make(chan rpcbus.Request, 1),
		getAddressChan:            make(chan rpcbus.Request, 1),
		getTxHistoryChan:          make(chan rpcbus.Request, 1),
		automateConsensusTxsChan:  make(chan rpcbus.Request, 1),
		isWalletLoadedChan:        make(chan rpcbus.Request, 1),
		clearWalletDatabaseChan:   make(chan rpcbus.Request, 1),
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
	if err := t.rb.Register(topics.LoadWallet, t.loadWalletChan); err != nil {
		return err
	}

	if err := t.rb.Register(topics.CreateWallet, t.createWalletChan); err != nil {
		return err
	}

	if err := t.rb.Register(topics.CreateFromSeed, t.createFromSeedChan); err != nil {
		return err
	}

	if err := t.rb.Register(topics.SendBidTx, t.sendBidTxChan); err != nil {
		return err
	}

	if err := t.rb.Register(topics.SendStakeTx, t.sendStakeTxChan); err != nil {
		return err
	}

	if err := t.rb.Register(topics.SendStandardTx, t.sendStandardTxChan); err != nil {
		return err
	}

	if err := t.rb.Register(topics.GetBalance, t.getBalanceChan); err != nil {
		return err
	}

	if err := t.rb.Register(topics.GetUnconfirmedBalance, t.getUnconfirmedBalanceChan); err != nil {
		return err
	}

	if err := t.rb.Register(topics.GetAddress, t.getAddressChan); err != nil {
		return err
	}

	if err := t.rb.Register(topics.GetTxHistory, t.getTxHistoryChan); err != nil {
		return err
	}

	if err := t.rb.Register(topics.AutomateConsensusTxs, t.automateConsensusTxsChan); err != nil {
		return err
	}

	if err := t.rb.Register(topics.IsWalletLoaded, t.isWalletLoadedChan); err != nil {
		return err
	}

	return t.rb.Register(topics.ClearWalletDatabase, t.clearWalletDatabaseChan)
}

// Wallet return wallet instance and err
func (t *Transactor) Wallet() (*wallet.Wallet, error) {
	if t.w == nil {
		return nil, errWalletNotLoaded
	}

	return t.w, nil
}

func (t *Transactor) launchMaintainer() error {
	if t.w == nil {
		return errWalletNotLoaded
	}

	if t.maintainerStarted {
		return errors.New("consensus transactions are already being automated")
	}

	k, err := t.w.ReconstructK()
	if err != nil {
		return err
	}

	log.Infof("maintainer is starting")
	m := maintainer.New(t.eb, t.rb, t.w.Keys().BLSPubKeyBytes, zkproof.CalculateM(k))
	go m.Listen()
	t.maintainerStarted = true
	return nil
}
