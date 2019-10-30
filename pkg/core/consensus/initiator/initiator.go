package initiator

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"time"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/factory"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/maintainer"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing/chainsync"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
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
	// TODO: sync first
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

	// Get current height
	req := rpcbus.NewRequest(bytes.Buffer{})
	resultBuf, err := rpcBus.Call(rpcbus.GetLastBlock, req, 1*time.Second)
	if err != nil {
		l.WithError(err).Warnln("could not retrieve current height, starting from 1")
		sendInitMessage(eventBroker, 1)
		return
	}

	var currentHeight uint64
	if err := encoding.ReadUint64LE(&resultBuf, &currentHeight); err != nil {
		l.WithError(err).Warnln("could not decode current height, starting from 1")
		sendInitMessage(eventBroker, 1)
		return
	}

	if currentHeight > 0 {
		// Get starting round
		startingRound := getStartingRound(eventBroker, counter)
		sendInitMessage(eventBroker, startingRound)
		return
	}

	// We are at genesis. Start from 1
	sendInitMessage(eventBroker, 1)
}

// XXX: clean this up ASAP
func startBlockGenerator(eventBroker eventbus.Broker, rpcBus *rpcbus.RPCBus, w *wallet.Wallet) {
	k, err := w.ReconstructK()
	if err != nil {
		panic(err)
	}

	m := zkproof.CalculateM(k)
	_, db := heavy.CreateDBConnection()
	for i := 0; ; i++ {
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

		var txs []transactions.Transaction
		err = db.View(func(t database.Transaction) error {
			var err error
			txs, err = t.FetchBlockTxs(hash)
			return err
		})
		if err != nil {
			panic(err)
		}

		for _, tx := range txs {
			bid, ok := tx.(*transactions.Bid)
			if !ok {
				continue
			}

			if bytes.Equal(bid.M, m.Bytes()) {
				err := db.Update(func(t database.Transaction) error {
					return t.SaveBidValues(bid.Outputs[0].Commitment.Bytes(), k.Bytes())
				})
				if err != nil {
					panic(err)
				}
			}
		}
	}
}

func getStartingRound(eventBroker eventbus.Broker, counter *chainsync.Counter) uint64 {
	// Start listening for accepted blocks, regardless of if we found stakes or not
	acceptedBlockChan, listener := consensus.InitAcceptedBlockUpdate(eventBroker)
	// Unsubscribe from AcceptedBlock once we're done
	defer listener.Quit()

	// Sync first
	syncToTip(acceptedBlockChan, counter)

	for {
		blk := <-acceptedBlockChan
		return blk.Header.Height + 1
	}
}

func sendInitMessage(publisher eventbus.Publisher, startingRound uint64) {

	l.Infof("Initialize consensus from starting round %d", startingRound)

	roundBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(roundBytes, startingRound)
	publisher.Publish(topics.Initialization, bytes.NewBuffer(roundBytes))

}

func syncToTip(acceptedBlockChan <-chan block.Block, counter *chainsync.Counter) {
	i := 0
	for {
		<-acceptedBlockChan
		if counter.IsSyncing() {
			i = 0
			continue
		}

		i++
		if i > 2 {
			break
		}
	}
}

func launchMaintainer(eventBroker eventbus.Broker, rpcBus *rpcbus.RPCBus, w *wallet.Wallet) error {
	r := cfg.Get()
	amount := r.Consensus.DefaultAmount
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
