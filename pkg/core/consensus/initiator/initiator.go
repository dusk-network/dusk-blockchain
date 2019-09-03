package initiator

import (
	"bytes"
	"encoding/binary"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/factory"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/generation"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/msg"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/wallet"
	log "github.com/sirupsen/logrus"
)

var l = log.WithField("process", "consensus initiator")

func LaunchConsensus(eventBroker wire.EventBroker, rpcBus *wire.RPCBus, w *wallet.Wallet) {
	go startProvisioner(eventBroker, rpcBus, w)
	go startBlockGenerator(eventBroker, rpcBus, w)
}

func startProvisioner(eventBroker wire.EventBroker, rpcBus *wire.RPCBus, w *wallet.Wallet) {
	// Setting up the consensus factory
	f := factory.New(eventBroker, rpcBus, config.ConsensusTimeOut, w.ConsensusKeys())
	f.StartConsensus()

	// Get starting round
	blsPubKey := w.ConsensusKeys().BLSPubKeyBytes
	startingRound := getStartingRound(blsPubKey, eventBroker)

	// Notify consensus components
	roundBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(roundBytes, startingRound)
	eventBroker.Publish(msg.InitializationTopic, bytes.NewBuffer(roundBytes))
}

func startBlockGenerator(eventBroker wire.EventBroker, rpcBus *wire.RPCBus, w *wallet.Wallet) {
	// make some random keys to sign the seed with
	keys, err := user.NewRandKeys()
	if err != nil {
		l.WithError(err).Warnln("could not start block generation component - problem generating keys")
		return
	}

	// reconstruct k
	k, err := reconstructK(w)
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

func getStartingRound(blsPubKey []byte, eventBroker wire.EventBroker) uint64 {
	// Start listening for accepted blocks, regardless of if we found stakes or not
	acceptedBlockChan, listener := consensus.InitAcceptedBlockUpdate(eventBroker)
	// Unsubscribe from AcceptedBlock once we're done
	defer listener.Quit()

	for {
		blk := <-acceptedBlockChan
		return blk.Header.Height + 1
	}
}

func reconstructK(w *wallet.Wallet) (ristretto.Scalar, error) {
	zeroPadding := make([]byte, 4)
	privSpend, err := w.PrivateSpend()
	if err != nil {
		return ristretto.Scalar{}, err
	}

	kBytes := append(privSpend, zeroPadding...)
	var k ristretto.Scalar
	k.Derive(kBytes)
	return k, nil
}
