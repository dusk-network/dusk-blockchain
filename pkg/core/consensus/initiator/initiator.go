package initiator

import (
	"bytes"
	"fmt"
	"os"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/factory"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/maintainer"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	"github.com/dusk-network/dusk-blockchain/pkg/core/marshalling"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing/chainsync"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-wallet/block"
	"github.com/dusk-network/dusk-wallet/transactions"
	"github.com/dusk-network/dusk-wallet/wallet"
	zkproof "github.com/dusk-network/dusk-zkproof"
	log "github.com/sirupsen/logrus"
)

var l = log.WithField("process", "consensus initiator")

func LaunchConsensus(eventBroker *eventbus.EventBus, rpcBus *rpcbus.RPCBus, w *wallet.Wallet, counter *chainsync.Counter) {
	startBlockGenerator(eventBroker, rpcBus, w)
	startProvisioner(eventBroker, rpcBus, w, counter)
	if err := launchMaintainer(eventBroker, rpcBus, w); err != nil {
		fmt.Fprintf(os.Stdout, "could not launch maintainer - consensus transactions will not be automated: %v\n", err)
	}
}

func startProvisioner(eventBroker *eventbus.EventBus, rpcBus *rpcbus.RPCBus, w *wallet.Wallet, counter *chainsync.Counter) {
	// Setting up the consensus factory
	pubKey := w.PublicKey()
	f := factory.New(eventBroker, rpcBus, cfg.ConsensusTimeOut, &pubKey, w.ConsensusKeys())
	f.StartConsensus()

	// If we are on genesis, we should kickstart the consensus
	lastBlkBuf, err := rpcBus.Call(rpcbus.GetLastBlock, rpcbus.Request{bytes.Buffer{}, make(chan rpcbus.Response, 1)}, 0)
	if err != nil {
		panic(err)
	}

	blk := block.NewBlock()
	if err := marshalling.UnmarshalBlock(&lastBlkBuf, blk); err != nil {
		panic(err)
	}

	if blk.Header.Height == 0 {
		eventBroker.Publish(topics.Initialization, new(bytes.Buffer))
	}
}

func startBlockGenerator(eventBroker eventbus.Broker, rpcBus *rpcbus.RPCBus, w *wallet.Wallet) {
	k, err := w.ReconstructK()
	if err != nil {
		panic(err)
	}

	m := zkproof.CalculateM(k)
	_, db := heavy.CreateDBConnection()
	for i := 0; ; i++ {
		// Get block hash for the given height
		var hash []byte
		err := db.View(func(t database.Transaction) error {
			var err error
			hash, err = t.FetchBlockHashByHeight(uint64(i))
			return err
		})

		// We hit the end of the chain, so we should just exit here
		if err != nil {
			return
		}

		// Get the transactions belonging to the previously found block
		var txs []transactions.Transaction
		err = db.View(func(t database.Transaction) error {
			var err error
			txs, err = t.FetchBlockTxs(hash)
			return err
		})
		if err != nil {
			panic(err)
		}

		// Check if we should store any of these transactions
		for _, tx := range txs {
			bid, ok := tx.(*transactions.Bid)
			if !ok {
				continue
			}

			if bytes.Equal(bid.M, m.Bytes()) {
				err := db.Update(func(t database.Transaction) error {
					return t.StoreBidValues(bid.Outputs[0].Commitment.Bytes(), k.Bytes())
				})
				if err != nil {
					panic(err)
				}
			}
		}
	}
}

func launchMaintainer(eventBroker eventbus.Broker, rpcBus *rpcbus.RPCBus, w *wallet.Wallet) error {
	r := cfg.Get()
	// default amount is denoted in whole units of DUSK, so we should convert it to
	// atomic units.
	amount := r.Consensus.DefaultAmount * wallet.DUSK
	lockTime := r.Consensus.DefaultLockTime
	if lockTime > transactions.MaxLockTime {
		log.Warnf("default locktime was configured to be greater than the maximum (%v) - defaulting to %v", lockTime, transactions.MaxLockTime)
		lockTime = transactions.MaxLockTime
	}

	offset := r.Consensus.DefaultOffset
	k, err := w.ReconstructK()
	if err != nil {
		return err
	}

	log.Infof("maintainer is starting with amount,locktime (%v,%v)", amount, lockTime)
	m, err := maintainer.New(eventBroker, rpcBus, w.ConsensusKeys().BLSPubKeyBytes, zkproof.CalculateM(k), amount, lockTime, offset)
	if err != nil {
		return err
	}
	go m.Listen()
	return nil
}
