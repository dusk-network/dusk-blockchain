package initiator

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/factory"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/generation"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/maintainer"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/msg"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/transactor"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing/chainsync"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/wallet"
	"github.com/dusk-network/dusk-blockchain/pkg/wallet/transactions"
	zkproof "github.com/dusk-network/dusk-zkproof"
	log "github.com/sirupsen/logrus"
)

var l = log.WithField("process", "consensus initiator")

func LaunchConsensus(eventBroker wire.EventBroker, rpcBus *wire.RPCBus, w *wallet.Wallet, counter *chainsync.Counter, transactor *transactor.Transactor) {
	// TODO: sync first
	go startProvisioner(eventBroker, rpcBus, w, counter)
	go startBlockGenerator(eventBroker, rpcBus, w)
	if err := launchMaintainer(eventBroker, transactor, w); err != nil {
		fmt.Fprintf(os.Stdout, "could not launch maintainer - consensus transactions will not be automated: %v\n", err)
	}
}

func startProvisioner(eventBroker wire.EventBroker, rpcBus *wire.RPCBus, w *wallet.Wallet, counter *chainsync.Counter) {
	// Setting up the consensus factory
	f := factory.New(eventBroker, rpcBus, config.ConsensusTimeOut, w.ConsensusKeys())
	f.StartConsensus()

	// Get current height
	req := wire.NewRequest(bytes.Buffer{}, 1)
	resultBuf, err := rpcBus.Call(wire.GetLastBlock, req)
	if err != nil {
		l.WithError(err).Warnln("could not retrieve current height, starting from 1")
		sendInitMessage(eventBroker, 1)
		return
	}

	var currentHeight uint64
	if err := encoding.ReadUint64(&resultBuf, binary.LittleEndian, &currentHeight); err != nil {
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

func startBlockGenerator(eventBroker wire.EventBroker, rpcBus *wire.RPCBus, w *wallet.Wallet) {
	// make some random keys to sign the seed with
	keys, err := user.NewRandKeys()
	if err != nil {
		l.WithError(err).Warnln("could not start block generation component - problem generating keys")
		return
	}

	// reconstruct k
	k, err := w.ReconstructK()
	if err != nil {
		l.WithError(err).Warnln("could not start block generation component - problem reconstructing K")
		return
	}

	// get public key that the rewards should go to
	publicKey := w.PublicKey()

	// launch generation component
	go func() {
		if err := generation.Launch(eventBroker, rpcBus, k, keys, &publicKey, nil, nil, nil); err != nil {
			l.WithError(err).Warnln("error launching block generation component")
		}
	}()
}

func getStartingRound(eventBroker wire.EventBroker, counter *chainsync.Counter) uint64 {
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

func sendInitMessage(publisher wire.EventPublisher, startingRound uint64) {
	roundBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(roundBytes, startingRound)
	publisher.Publish(msg.InitializationTopic, bytes.NewBuffer(roundBytes))
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

func launchMaintainer(eventBroker wire.EventBroker, transactor *transactor.Transactor, w *wallet.Wallet) error {
	r := cfg.Get()
	amount := r.Consensus.DefaultAmount
	lockTime := r.Consensus.DefaultLockTime
	if lockTime > transactions.MaxLockTime {
		fmt.Fprintf(os.Stdout, "default locktime was configured to be greater than the maximum (%v) - defaulting to %v\n", lockTime, transactions.MaxLockTime)
		lockTime = transactions.MaxLockTime
	}

	offset := r.Consensus.DefaultOffset
	k, err := w.ReconstructK()
	if err != nil {
		return err
	}

	return maintainer.Launch(eventBroker, nil, w.ConsensusKeys().BLSPubKeyBytes, zkproof.CalculateM(k), transactor, amount, lockTime, offset)
}
