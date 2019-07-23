package cli

import (
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"os"
	"sort"
	"strconv"

	ristretto "github.com/bwesterb/go-ristretto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/factory"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/generation"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database/heavy"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
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
	"sync":                syncWalletCMD,
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

	if err := consensus.GetStartingRound(publisher, nil, cliWallet.ConsensusKeys()); err != nil {
		fmt.Fprintf(os.Stdout, "error starting consensus: %v\n", err)
		return
	}

	fmt.Fprintf(os.Stdout, "provisioner module started\nto more accurately follow the progression of consensus, use the showlogs command\n")
}

func startBlockGenerator(args []string, publisher wire.EventBroker, rpcBus *wire.RPCBus) {
	if len(args) == 0 || args[0] == "" {
		fmt.Fprintf(os.Stdout, "please provide a bidding tx hash to start block generation\n")
		return
	}

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

	txID, err := hex.DecodeString(args[0])
	if err != nil {
		fmt.Fprintf(os.Stdout, "error attempting to decode tx: %v\n", err)
		return
	}

	// fetch bid tx
	_, db := heavy.CreateDBConnection()
	var tx transactions.Transaction
	err = db.View(func(t database.Transaction) error {
		var err error
		tx, _, _, err = t.FetchBlockTxByHash(txID)
		return err
	})

	if err != nil {
		fmt.Fprintf(os.Stdout, "could not find supplied bid tx\n")
		return
	}

	bid, ok := tx.(*transactions.Bid)
	if !ok {
		fmt.Fprintf(os.Stdout, "supplied txID does not point to a bidding transaction\n")
		return
	}

	var d ristretto.Scalar
	d.UnmarshalBinary(bid.Outputs[0].Commitment)

	// launch generation component
	publicKey := cliWallet.PublicKey()
	generation.Launch(publisher, rpcBus, d, k, nil, nil, keys, &publicKey)

	fmt.Fprintf(os.Stdout, "block generator module started\nto more accurately follow the progression of consensus, use the showlogs command\n")
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
