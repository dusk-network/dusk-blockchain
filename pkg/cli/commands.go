package cli

import (
	"bytes"
	"fmt"
	"math/big"
	"os"
	"sort"
	"strconv"

	ristretto "github.com/bwesterb/go-ristretto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

// CLICommands holds all of the wallet commands that the user can call through
// the interactive shell.
var CLICommands = map[string]func([]string, wire.EventBroker){
	"help":           showHelp,
	"createwallet":   createWalletCMD,
	"loadwallet":     loadWalletCMD,
	"transfer":       transferCMD,
	"stake":          sendStakeCMD,
	"bid":            sendBidCMD,
	"sync":           syncWalletCMD,
	"startconsensus": startConsensus,
	"exit":           stopNode,
	"quit":           stopNode,
}

func showHelp(args []string, publisher wire.EventBroker) {
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

func startConsensus(args []string, publisher wire.EventBroker) {
	publisher.Publish(string(topics.StartConsensus), new(bytes.Buffer))
}

func stopNode(args []string, publisher wire.EventBroker) {
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
	sInt, err := strconv.Atoi(s)
	if err != nil {
		return ristretto.Scalar{}, err
	}
	return intToScalar(int64(sInt)), nil
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
