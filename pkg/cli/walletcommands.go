package cli

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
)

// commandLineProcessor interprets the CLI commands
type commandLineProcessor struct {
	rpcBus *rpcbus.RPCBus
}

func (c *commandLineProcessor) createWalletCMD(args []string) {
	if args == nil || len(args) < 1 {
		fmt.Fprintf(os.Stdout, commandInfo["createwallet"]+"\n")
		return
	}
	password := args[0]

	pubKey, err := c.createWallet(password)
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

	pubKey, err := c.loadWallet(args[0])
	if err != nil {
		fmt.Fprintf(os.Stdout, "error creating wallet: %v\n", err)
		return
	}

	fmt.Fprintf(os.Stdout, "Wallet loaded successfully!\n")
	fmt.Fprintf(os.Stdout, "Public Address: %s\n", pubKey)
}

func (c *commandLineProcessor) createFromSeedCMD(args []string) {
	if args == nil || len(args) < 2 {
		fmt.Fprintf(os.Stdout, commandInfo["createfromseed"]+"\n")
		return
	}

	seed := args[0]
	password := args[1]

	pubKey, err := c.createFromSeed(seed, password)
	if err != nil {
		fmt.Fprintf(os.Stdout, "error creating wallet from seed: %v\n", err)
		return
	}

	fmt.Fprintf(os.Stdout, "Wallet created successfully!\n")
	fmt.Fprintf(os.Stdout, "Public Address: %s\n", pubKey)
}

func (c *commandLineProcessor) balanceCMD() {
	walletBalance, mempoolBalance, err := c.getBalance()
	if err != nil {
		fmt.Fprintf(os.Stdout, "error retrieving balance: %v\n", err)
		return
	}

	// unlocked balance is the amount of outputs currently available to spend
	unlockedBalance := float64(walletBalance) / float64(cfg.DUSK)
	// overall balance is sum of the unlockedBalance plus pending to be received
	overallBalance := float64(walletBalance+mempoolBalance) / float64(cfg.DUSK)

	fmt.Fprintf(os.Stdout, "Balance %.8f, Unlocked Balance %.8f\n", overallBalance, unlockedBalance)
}

func (c *commandLineProcessor) transferCMD(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stdout, "Please specify an amount and an address")
		return
	}

	amount, err := parseAmountValue(args[0])
	if err != nil {
		fmt.Fprintf(os.Stdout, "error converting amount: %v\n", err)
		return
	}

	pubKey := args[1]

	txid, err := c.sendStandardTx(amount, pubKey)
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

	amount, err := parseAmountValue(args[0])
	if err != nil {
		fmt.Fprintf(os.Stdout, "error converting amount: %v\n", err)
		return
	}

	lockTime, err := stringToUint64(args[1])
	if err != nil {
		fmt.Fprintf(os.Stdout, "error converting locktime string to an integer: %v\n", err)
		return
	}

	txid, err := c.sendBidTx(amount, lockTime)
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

	amount, err := parseAmountValue(args[0])
	if err != nil {
		fmt.Fprintf(os.Stdout, "error converting amount: %v\n", err)
		return
	}

	lockTime, err := stringToUint64(args[1])
	if err != nil {
		fmt.Fprintf(os.Stdout, "error converting locktime string to an integer: %v\n", err)
		return
	}

	txid, err := c.sendStakeTx(amount, lockTime)
	if err != nil {
		fmt.Fprintf(os.Stdout, "error: %v\n", err)
		return
	}

	fmt.Fprintf(os.Stdout, "Txn Hash: %s\n", hex.EncodeToString(txid))
}

// parseAmountValue convert DUSK amount value into atomic units where
// 1 atomic unit is 0.00000001 DUSK
func parseAmountValue(value string) (uint64, error) {
	amountInDusk, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, err
	}

	// convert to DUSK atomic units
	amountInUnits := amountInDusk * float64(cfg.DUSK)
	return uint64(amountInUnits), nil
}

func (c *commandLineProcessor) getBalance() (uint64, uint64, error) {
	buf := new(bytes.Buffer)
	req := rpcbus.NewRequest(*buf)
	resultBuf, err := c.rpcBus.Call(rpcbus.GetBalance, req, 0)
	if err != nil {
		return 0, 0, err
	}

	var walletBalance uint64
	if err := encoding.ReadUint64LE(&resultBuf, &walletBalance); err != nil {
		return 0, 0, err
	}

	var mempoolBalance uint64
	if err := encoding.ReadUint64LE(&resultBuf, &mempoolBalance); err != nil {
		return walletBalance, 0, err
	}

	return walletBalance, mempoolBalance, nil
}

func (c *commandLineProcessor) sendStandardTx(amount uint64, pubkey string) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := encoding.WriteUint64LE(buf, amount); err != nil {
		return nil, err
	}

	if err := encoding.WriteString(buf, pubkey); err != nil {
		return nil, err
	}

	req := rpcbus.NewRequest(*buf)
	txIdBuf, err := c.rpcBus.Call(rpcbus.SendStandardTx, req, 0)
	if err != nil {
		return nil, err
	}

	return txIdBuf.Bytes(), nil
}

func (c *commandLineProcessor) loadWallet(password string) (string, error) {
	buf := new(bytes.Buffer)
	if err := encoding.WriteString(buf, password); err != nil {
		return "", err
	}

	req := rpcbus.NewRequest(*buf)
	pubKeyBuf, err := c.rpcBus.Call(rpcbus.LoadWallet, req, 0)
	if err != nil {
		return "", err
	}

	var pubKey string
	pubKey, err = encoding.ReadString(&pubKeyBuf)
	if err != nil {
		return "", err
	}

	return pubKey, nil
}

func (c *commandLineProcessor) createFromSeed(seed, password string) (string, error) {
	buf := new(bytes.Buffer)
	if err := encoding.WriteString(buf, seed); err != nil {
		return "", err
	}

	if err := encoding.WriteString(buf, password); err != nil {
		return "", err
	}

	req := rpcbus.NewRequest(*buf)
	pubKeyBuf, err := c.rpcBus.Call(rpcbus.CreateFromSeed, req, 0)
	if err != nil {
		return "", err
	}

	var pubKey string
	pubKey, err = encoding.ReadString(&pubKeyBuf)
	if err != nil {
		return "", err
	}

	return pubKey, nil
}

func (c *commandLineProcessor) createWallet(password string) (string, error) {
	buf := new(bytes.Buffer)
	if err := encoding.WriteString(buf, password); err != nil {
		return "", err
	}

	req := rpcbus.NewRequest(*buf)
	pubKeyBuf, err := c.rpcBus.Call(rpcbus.CreateWallet, req, 0)
	if err != nil {
		return "", err
	}

	var pubKey string
	pubKey, err = encoding.ReadString(&pubKeyBuf)
	if err != nil {
		return "", err
	}

	return pubKey, nil
}

func (c *commandLineProcessor) sendBidTx(amount, lockTime uint64) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := rpcbus.MarshalConsensusTxRequest(buf, amount, lockTime); err != nil {
		return nil, err
	}

	req := rpcbus.NewRequest(*buf)
	txIdBuf, err := c.rpcBus.Call(rpcbus.SendBidTx, req, 0)
	if err != nil {
		return nil, err
	}

	return txIdBuf.Bytes(), nil
}

func (c *commandLineProcessor) sendStakeTx(amount, lockTime uint64) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := rpcbus.MarshalConsensusTxRequest(buf, amount, lockTime); err != nil {
		return nil, err
	}

	req := rpcbus.NewRequest(*buf)
	txIdBuf, err := c.rpcBus.Call(rpcbus.SendStakeTx, req, 0)
	if err != nil {
		return nil, err
	}

	return txIdBuf.Bytes(), nil
}
