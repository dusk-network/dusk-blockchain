package cli

import (
	"fmt"
	"math"
	"math/big"
	"os"
	"sort"
	"strconv"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
)

// CLICommands holds all of the wallet commands that the user can call through
// the interactive shell.
var CLICommands = map[string]func([]string, wire.EventBroker, *wire.RPCBus){
	"help":               showHelp,
	"createwallet":       createWalletCMD,
	"loadwallet":         loadWalletCMD,
	"createfromseed":     createFromSeedCMD,
	"balance":            balanceCMD,
	"transfer":           transferCMD,
	"stake":              sendStakeCMD,
	"bid":                sendBidCMD,
	"setdefaultlocktime": setLocktimeCMD,
	"setdefaultvalue":    setDefaultValueCMD,
	"exit":               stopNode,
	"quit":               stopNode,
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

func setLocktimeCMD(args []string, publisher wire.EventBroker, rpcBus *wire.RPCBus) {
	if args == nil || len(args) < 1 {
		fmt.Fprintln(os.Stdout, "please specify a default locktime")
		return
	}

	locktime, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Fprintln(os.Stdout, err)
		return
	}

	if locktime > 250000 {
		fmt.Fprintln(os.Stdout, "locktime can not be higher than 250000")
	}

	// Send value to tx sender
	fmt.Println(locktime)
}

func setDefaultValueCMD(args []string, publisher wire.EventBroker, rpcBus *wire.RPCBus) {
	if args == nil || len(args) < 1 {
		fmt.Fprintln(os.Stdout, "please specify a default value")
		return
	}

	value, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Fprintln(os.Stdout, err)
		return
	}

	// Send value to tx sender
	fmt.Println(value)
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
