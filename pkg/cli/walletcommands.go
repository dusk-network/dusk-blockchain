package cli

import (
	"encoding/hex"
	"fmt"
	"os"

	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
)

// commandLineProcessor interpretes the CLI commands
type commandLineProcessor struct {
	rpcBus *rpcbus.RPCBus
}

func (c *commandLineProcessor) createWalletCMD(args []string) {

	if args == nil || len(args) < 1 {
		fmt.Fprintf(os.Stdout, commandInfo["createwallet"]+"\n")
		return
	}
	password := args[0]

	pubKey, err := c.rpcBus.CreateWallet(password)
	if err != nil {
		fmt.Fprintf(os.Stdout, "error creating wallet from seed: %v\n", err)
		return
	}

	fmt.Fprintf(os.Stdout, "Wallet created successfully!\n")
	fmt.Fprintf(os.Stdout, "Public Address: %s\n", pubKey)

}

func (c *commandLineProcessor) loadWalletCMD(args []string) {

	if args == nil || len(args) < 1 {
		fmt.Fprintf(os.Stdout, commandInfo["loadwallet"]+"\n")
		return
	}

	pubKey, err := c.rpcBus.LoadWallet(args[0])
	if err != nil {
		fmt.Fprintf(os.Stdout, "error creating wallet: %v\n", err)
		return
	}

	fmt.Fprintf(os.Stdout, "Wallet created successfully!\n")
	fmt.Fprintf(os.Stdout, "Public Address: %s\n", pubKey)

}

func (c *commandLineProcessor) createFromSeedCMD(args []string) {

	if args == nil || len(args) < 2 {
		fmt.Fprintf(os.Stdout, commandInfo["createfromseed"]+"\n")
		return
	}

	seed := args[0]
	password := args[1]

	pubKey, err := c.rpcBus.CreateFromSeed(seed, password)
	if err != nil {
		fmt.Fprintf(os.Stdout, "error creating wallet from seed: %v\n", err)
		return
	}

	fmt.Fprintf(os.Stdout, "Wallet loaded successfully!\n")
	fmt.Fprintf(os.Stdout, "Public Address: %s\n", pubKey)
}

func (c *commandLineProcessor) balanceCMD() {

	balance, err := c.rpcBus.GetBalance()
	if err != nil {
		fmt.Fprintf(os.Stdout, "error retrieving balance: %v\n", err)
		return
	}

	fmt.Fprintln(os.Stdout, balance)
}

func (c *commandLineProcessor) transferCMD(args []string) {

	if len(args) < 2 {
		fmt.Fprintln(os.Stdout, "Please specify an amount and an address")
		return
	}

	amount, err := stringToUint64(args[0])
	if err != nil {
		fmt.Fprintf(os.Stdout, "error converting amount string to an integer: %v\n", err)
		return
	}

	pubKey := args[1]

	txid, err := c.rpcBus.SendStandardTx(amount, pubKey)
	if err != nil {
		fmt.Fprintf(os.Stdout, "error: %v\n", err)
		return
	}

	fmt.Fprintf(os.Stdout, "Txn Hash: %s\n", hex.EncodeToString(txid))
}

func (c *commandLineProcessor) sendBidCMD(args []string) {

	if len(args) < 2 {
		fmt.Fprintln(os.Stdout, "Please specify an amount and lock time")
		return
	}

	amount, err := stringToUint64(args[0])
	if err != nil {
		fmt.Fprintf(os.Stdout, "error converting amount string to an integer: %v\n", err)
		return
	}

	lockTime, err := stringToUint64(args[1])
	if err != nil {
		fmt.Fprintf(os.Stdout, "error converting locktime string to an integer: %v\n", err)
		return
	}

	txid, err := c.rpcBus.SendBidTx(amount, lockTime)
	if err != nil {
		fmt.Fprintf(os.Stdout, "error: %v\n", err)
		return
	}

	fmt.Fprintf(os.Stdout, "Txn Hash: %s\n", hex.EncodeToString(txid))
}

func (c *commandLineProcessor) sendStakeCMD(args []string) {

	if len(args) < 2 {
		fmt.Fprintln(os.Stdout, "Please specify an amount and lock time")
		return
	}

	amount, err := stringToUint64(args[0])
	if err != nil {
		fmt.Fprintf(os.Stdout, "error converting amount string to an integer: %v\n", err)
		return
	}

	lockTime, err := stringToUint64(args[1])
	if err != nil {
		fmt.Fprintf(os.Stdout, "error converting locktime string to an integer: %v\n", err)
		return
	}

	txid, err := c.rpcBus.SendStakeTx(amount, lockTime)
	if err != nil {
		fmt.Fprintf(os.Stdout, "error: %v\n", err)
		return
	}

	fmt.Fprintf(os.Stdout, "Txn Hash: %s\n", hex.EncodeToString(txid))
}
