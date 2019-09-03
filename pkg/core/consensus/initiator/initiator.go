package initiator

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/factory"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/generation"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/msg"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/wallet"
)

func LaunchConsensus(publisher wire.EventPublisher, rpcBus *wire.RPCBus, w *wallet.Wallet) {
	go startProvisioner(publisher, rpcBus, w)
	go startBlockGenerator(publisher, rpcBus, w)
}

func startProvisioner(publisher wire.EventPublisher, rpcBus *wire.RPCBus, w *wallet.Wallet) {
	// Setting up the consensus factory
	f := factory.New(publisher, rpcBus, config.ConsensusTimeOut, w.ConsensusKeys())
	f.StartConsensus()

	blsPubKey := w.ConsensusKeys().BLSPubKeyBytes

	startingRound := getStartingRound(blsPubKey, publisher)

	// Notify consensus components
	roundBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(roundBytes, startingRound)
	publisher.Publish(msg.InitializationTopic, bytes.NewBuffer(roundBytes))
}

func startBlockGenerator(publisher wire.EventPublisher, rpcBus *wire.RPCBus, w *wallet.Wallet) {
	// make some random keys to sign the seed with
	keys, err := user.NewRandKeys()
	if err != nil {
		fmt.Fprintf(os.Stdout, "error starting block generation component - could not generate keys: %v\n", err)
		return
	}

	// reconstruct k
	zeroPadding := make([]byte, 4)
	privSpend, err := w.PrivateSpend()
	if err != nil {
		fmt.Fprintf(os.Stdout, "error starting block generation component - could not get private spend: %v\n", err)
		return
	}

	kBytes := append(privSpend, zeroPadding...)
	var k ristretto.Scalar
	k.Derive(kBytes)

	// get public key that the rewards should go to
	publicKey := w.PublicKey()

	// launch generation component
	go func() {
		if err := generation.Launch(publisher, rpcBus, k, keys, &publicKey, nil, nil, nil); err != nil {
			fmt.Fprintf(os.Stdout, "error launching block generation component: %v\n", err)
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
