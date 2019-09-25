package cli

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing/chainsync"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
)

var testnet = byte(2)

// TODO: rename
type CLI struct {
	eventBroker wire.EventBroker
	rpcBus      *wire.RPCBus
	counter     *chainsync.Counter
}

func (c *CLI) createWalletCMD(args []string) {

	/*
		if args == nil || len(args) < 1 {
			fmt.Fprintf(os.Stdout, commandInfo["createwallet"]+"\n")
			return
		}
		password := args[0]

		pBuf := bytes.Newbuffer([]byte(password))
		req := wire.NewRequest(pBuf, 2)
		c.rpcBus.Call(wire.CreateWallet, req)

		select {
		case addrBuf := <-req.RespChan:
			pubAddr := string(addrBuf.Bytes())
			fmt.Fprintf(os.Stdout, "Wallet created successfully!\n")
			fmt.Fprintf(os.Stdout, "Public Address: %s\n", pubAddr)

			if !cfg.Get().General.WalletOnly {
				initiator.LaunchConsensus(c.eventBroker, c.rpcBus, w, c.counter, c.transactor)
			}
		case err := <-req.ErrChan:
			fmt.Fprintf(os.Stdout, "error creating wallet: %v\n", err)
		}
	*/
}

func (c *CLI) loadWalletCMD(args []string) {

	if args == nil || len(args) < 1 {
		fmt.Fprintf(os.Stdout, commandInfo["loadwallet"]+"\n")
		return
	}

	pubKey, err := loadWallet(c.rpcBus, args[0])
	if err != nil {
		fmt.Fprintf(os.Stdout, "error creating wallet: %v\n", err)
		return
	}

	fmt.Fprintf(os.Stdout, "Wallet created successfully!\n")
	fmt.Fprintf(os.Stdout, "Public Address: %s\n", pubKey)

}

func (c *CLI) createFromSeedCMD(args []string) {

	/*
		if args == nil || len(args) < 2 {
			fmt.Fprintf(os.Stdout, commandInfo["createfromseed"]+"\n")
			return
		}

		seed := args[0]
		password := args[1]
		seedBytes, err := hex.DecodeString(seed)
		if err != nil {
			fmt.Fprintf(os.Stdout, "error attempting to decode seed: %v\n", err)
			return
		}

		// Then load the wallet
		w, err := createFromSeed(seedBytes, password)
		if err != nil {
			fmt.Fprintf(os.Stdout, "error attempting to create wallet from seed: %v\n", err)
			return
		}

		pubAddr, err := w.PublicAddress()
		if err != nil {
			fmt.Fprintf(os.Stdout, "error attempting to get your public address: %v\n", err)
			return
		}

		fmt.Fprintf(os.Stdout, "Wallet loaded successfully!\nPublic Address: %s\n", pubAddr)

		c.transactor = transactor.New(w, nil)

		if !cfg.Get().General.WalletOnly {
			initiator.LaunchConsensus(c.eventBroker, c.rpcBus, w, c.counter, c.transactor)
		}
	*/
}

func (c *CLI) balanceCMD() {

	/*
		balance, err := c.transactor.Balance()
		if err != nil {
			fmt.Fprintf(os.Stdout, "error retrieving balance: %v\n", err)
			return
		}
		fmt.Fprintln(os.Stdout, balance)
	*/
}

func (c *CLI) transferCMD(args []string) {

	/*
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

		// TODO:
	*/
}

func (c *CLI) sendBidCMD(args []string) {

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

	txid, err := sendBidTx(c.rpcBus, amount, lockTime)
	if err != nil {
		fmt.Fprintf(os.Stdout, "error: %v\n", err)
		return
	}

	fmt.Fprintf(os.Stdout, "BidTx ID: %s\n", txid)
}

func (c *CLI) sendStakeCMD(args []string) {

	/*
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

		// TODO:
	*/
}

func (c *CLI) sendTransaction(methodName string, amount, locktime uint64, pubKey string) {

	pBuf := bytes.NewBuffer([]byte(""))
	req := wire.NewRequest(*pBuf, -1)

	result, err := c.rpcBus.Call(methodName, req)
	if err != nil {
		fmt.Fprintf(os.Stdout, "error on %d call %s", err)
	}

	hash := hex.EncodeToString(result.Bytes())
	fmt.Fprintf(os.Stdout, "txid: %s", hash)
}

func loadWallet(rb *wire.RPCBus, password string) (string, error) {

	pBuf := bytes.NewBufferString(password)
	req := wire.NewRequest(*pBuf, -1)
	pubKeyBuf, err := rb.Call(wire.LoadWallet, req)
	if err != nil {
		return "", err
	}

	return pubKeyBuf.String(), nil
}

func sendBidTx(rb *wire.RPCBus, amount, lockTime uint64) (string, error) {

	buf := new(bytes.Buffer)

	if err := encoding.WriteUint64(buf, binary.LittleEndian, amount); err != nil {
		return "", err
	}

	if err := encoding.WriteUint64(buf, binary.LittleEndian, lockTime); err != nil {
		return "", err
	}

	req := wire.NewRequest(*buf, -1)
	txIdBuf, err := rb.Call(wire.SendBidTx, req)
	if err != nil {
		return "", err
	}

	return txIdBuf.String(), nil
}
