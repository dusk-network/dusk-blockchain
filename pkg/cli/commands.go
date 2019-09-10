package cli

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"os"
	"sort"
	"strconv"
	"time"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/factory"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/generation"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/msg"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
)

// CLICommands holds all of the wallet commands that the user can call through
// the interactive shell.
var CLICommands = map[string]func([]string, wire.EventBroker, *wire.RPCBus){
	"help":                showHelp,
	"createwallet":        createWalletCMD,
	"loadwallet":          loadWalletCMD,
	"createfromseed":      createFromSeedCMD,
	"balance":             balanceCMD,
	"transfer":            transferCMD,
	"stake":               sendStakeCMD,
	"bid":                 sendBidCMD,
	"startprovisioner":    startProvisioner,
	"startblockgenerator": startBlockGenerator,
	"exit":                stopNode,
	"quit":                stopNode,
}

func showHelp(args []string, publisher wire.EventBroker, rpcBus *wire.RPCBus) {
	if args != nil && len(args) > 0 {
		helpStr, ok := commandInfo[args[0]]
		if !ok {
			fmt.Fprintf(os.Stdout, "%v is not a supported command\n", args[0])
			return
		}

		fmt.Fprintln(os.Stdout, helpStr)
		return
	}

	commands := make([]string, 0)
	for cmd, desc := range commandInfo {
		commands = append(commands, cmd+": "+desc)
	}

	sort.Strings(commands)
	for _, cmd := range commands {
		fmt.Fprintln(os.Stdout, "\t"+cmd+"\n")
	}
}

func startProvisioner(args []string, publisher wire.EventBroker, rpcBus *wire.RPCBus) {
	// load consensus info, get keys, D and K
	if cliWallet == nil {
		fmt.Fprintf(os.Stdout, "please load a wallet before trying to participate in consensus\n")
		return
	}

	// Setting up the consensus factory
	f := factory.New(publisher, rpcBus, config.ConsensusTimeOut, cliWallet.ConsensusKeys())
	f.StartConsensus()

	blsPubKey := cliWallet.ConsensusKeys().BLSPubKeyBytes

	time.Sleep(2 * time.Second)
	// TODO: Rework when dusk-blockchain/issues/27 is done
	roundBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(roundBytes, 1)
	publisher.Publish(msg.InitializationTopic, bytes.NewBuffer(roundBytes))

	go func() {
		startingRound := getStartingRound(blsPubKey, publisher)

		// Notify consensus components
		roundBytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(roundBytes, startingRound)
		publisher.Publish(msg.InitializationTopic, bytes.NewBuffer(roundBytes))
	}()

	fmt.Fprintf(os.Stdout, "provisioner module started\nto more accurately follow the progression of consensus, use the showlogs command\n")
}

func startBlockGenerator(args []string, publisher wire.EventBroker, rpcBus *wire.RPCBus) {
	if cliWallet == nil {
		fmt.Fprintf(os.Stdout, "please load a wallet before trying to participate in consensus\n")
		return
	}

	// make some random keys to sign the seed with
	keys, err := user.NewRandKeys()
	if err != nil {
		fmt.Fprintf(os.Stdout, "could not generate keys: %v\n", err)
		return
	}

	// reconstruct k
	zeroPadding := make([]byte, 4)
	privSpend, err := cliWallet.PrivateSpend()
	if err != nil {
		fmt.Fprintf(os.Stdout, "could not get private spend: %v\n", err)
		return
	}

	kBytes := append(privSpend, zeroPadding...)
	var k ristretto.Scalar
	k.Derive(kBytes)

	// get public key that the rewards should go to
	publicKey := cliWallet.PublicKey()

	// launch generation component
	go func() {
		if err := generation.Launch(publisher, rpcBus, k, keys, &publicKey, nil, nil, nil); err != nil {
			fmt.Fprintf(os.Stdout, "error launching block generation component: %v\n", err)
		}
	}()

	fmt.Fprintf(os.Stdout, "block generation component started\nto more accurately follow the progression of consensus, use the showlogs command\n")
}

func stopNode(args []string, publisher wire.EventBroker, rpcBus *wire.RPCBus) {
	fmt.Fprintln(os.Stdout, "stopping node")

	// Send an interrupt signal to the running process
	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		// This should never ever happen
		panic(err)
	}

	if err := p.Signal(os.Interrupt); err != nil {
		// Neither should this
		panic(err)
	}
}

func intToScalar(amount int64) ristretto.Scalar {
	var x ristretto.Scalar
	x.SetBigInt(big.NewInt(amount))
	return x
}

func stringToScalar(s string) (ristretto.Scalar, error) {
	sFloat, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return ristretto.Scalar{}, err
	}

	sInt := int64(math.Floor(sFloat * float64(config.DUSK)))
	return intToScalar(sInt), nil
}

func stringToInt64(s string) (int64, error) {
	sInt, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	return (int64(sInt)), nil
}

func stringToUint64(s string) (uint64, error) {
	sInt, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	return (uint64(sInt)), nil
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
