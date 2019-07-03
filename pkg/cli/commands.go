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
	"gitlab.dusk.network/dusk-core/dusk-wallet/database"
	"gitlab.dusk.network/dusk-core/dusk-wallet/key"
	wallet "gitlab.dusk.network/dusk-core/dusk-wallet/wallet/v3"
)

// CLICommands holds all of the wallet commands that the user can call through
// the interactive shell.
var CLICommands = map[string]func([]string, wire.EventPublisher){
	"help":           showHelp,
	"createwallet":   createWallet,
	"transfer":       transfer,
	"stake":          sendStake,
	"bid":            sendBid,
	"startconsensus": startConsensus,
	"exit":           stopNode,
	"quit":           stopNode,
}

func showHelp(args []string, publisher wire.EventPublisher) {
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

func createWallet(args []string, publisher wire.EventPublisher) {
	// TODO: add proper path
	db, err := database.New("")
	if err != nil {
		// TODO: use logger over fmt.Print
		fmt.Fprintf(os.Stdout, "error opening database: %v\n", err)
		return
	}

	// TODO: add functions
	w, err := wallet.New(2, db, nil, nil)
	if err != nil {
		fmt.Fprintf(os.Stdout, "error creating wallet: %v\n", err)
		return
	}

	if err := w.Save(); err != nil {
		fmt.Fprintf(os.Stdout, "error saving wallet: %v\n", err)
	}
}

func transfer(args []string, publisher wire.EventPublisher) {
	if args == nil || len(args) < 2 {
		fmt.Fprintf(os.Stdout, commandInfo["transfer"])
		return
	}

	w, err := wallet.LoadWallet()
	if err != nil {
		fmt.Fprintf(os.Stdout, "error opening wallet: %v\n", err)
		return
	}

	// TODO: decide on fee in a proper way
	fee := 100
	if len(args) == 3 {
		fee, err = strconv.Atoi(args[2])
		if err != nil {
			fmt.Fprintf(os.Stdout, "fee should be specified as a number\n")
			return
		}
	}

	tx, err := w.NewStandardTx(int64(fee))
	if err != nil {
		fmt.Fprintf(os.Stdout, "error creating tx: %v\n", err)
		return
	}

	amount, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Fprintf(os.Stdout, "amount should be a number\n")
		return
	}

	tx.AddOutput(key.PublicAddress(args[1]), intToScalar(int64(amount)))
	buf := new(bytes.Buffer)
	if err := tx.Encode(buf); err != nil {
		fmt.Fprintf(os.Stdout, "error encoding tx: %v\n", err)
		return
	}

	publisher.Publish(string(topics.Tx), buf)
}

func sendStake(args []string, publisher wire.EventPublisher) {
	if args == nil || len(args) < 1 {
		fmt.Fprintf(os.Stdout, commandInfo["stake"])
		return
	}

	_, err := wallet.LoadWallet()
	if err != nil {
		fmt.Fprintf(os.Stdout, "error opening wallet: %v\n", err)
		return
	}
}

func sendBid(args []string, publisher wire.EventPublisher) {
	if args == nil || len(args) < 1 {
		fmt.Fprintf(os.Stdout, commandInfo["bid"])
		return
	}

	_, err := wallet.LoadWallet()
	if err != nil {
		fmt.Fprintf(os.Stdout, "error opening wallet: %v\n", err)
		return
	}
}

func startConsensus(args []string, publisher wire.EventPublisher) {
	publisher.Publish(string(topics.StartConsensus), new(bytes.Buffer))
}

func stopNode(args []string, publisher wire.EventPublisher) {
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
