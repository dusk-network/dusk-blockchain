package cli

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/initiator"
	"github.com/dusk-network/dusk-blockchain/pkg/core/transactor"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing/chainsync"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/wallet"
	walletdb "github.com/dusk-network/dusk-blockchain/pkg/wallet/database"

	"github.com/dusk-network/dusk-blockchain/pkg/wallet/transactions"
)

var testnet = byte(2)

// TODO: rename
type CLI struct {
	eventBroker wire.EventBroker
	rpcBus      *wire.RPCBus
	counter     *chainsync.Counter
}

func (c *CLI) createWalletCMD(args []string) {
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
}

func (c *CLI) loadWalletCMD(args []string) {
	if args == nil || len(args) < 1 {
		fmt.Fprintf(os.Stdout, commandInfo["loadwallet"]+"\n")
		return
	}
	password := args[0]

	pBuf := bytes.Newbuffer([]byte(password))
	req := wire.NewRequest(pBuf, 2)
	c.rpcBus.Call(wire.LoadWallet, req)

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
}

func loadWallet(password string) (*wallet.Wallet, error) {
	// First load the database
	db, err := walletdb.New(cfg.Get().Wallet.Store)
	if err != nil {
		db.Close()
		return nil, err
	}

	// Then load the wallet
	w, err := wallet.LoadFromFile(testnet, db, fetchDecoys, fetchInputs, password)
	if err != nil {
		db.Close()
		return nil, err
	}

	return w, nil
}

func (c *CLI) createFromSeedCMD(args []string) {
	if c.transactor != nil {
		fmt.Fprintln(os.Stdout, "You have already loaded a wallet. Please re-start the node to load another.")
	}

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
}

func createFromSeed(seedBytes []byte, password string) (*wallet.Wallet, error) {
	// First load the database
	db, err := walletdb.New(cfg.Get().Wallet.Store)
	if err != nil {
		return nil, err
	}

	// Then load the wallet
	w, err := wallet.LoadFromSeed(seedBytes, testnet, db, fetchDecoys, fetchInputs, password)
	if err != nil {
		return nil, err
	}

	return w, nil
}

func (c *CLI) balanceCMD() {
	if c.transactor == nil {
		fmt.Fprintln(os.Stdout, "Please load a wallet before checking your balance")
		return
	}

	balance, err := c.transactor.Balance()
	if err != nil {
		fmt.Fprintf(os.Stdout, "error retrieving balance: %v\n", err)
		return
	}
	fmt.Fprintln(os.Stdout, balance)
}

func (c *CLI) transferCMD(args []string) {
	if c.transactor == nil {
		fmt.Fprintln(os.Stdout, "Please load a wallet before sending DUSK")
		return
	}

	if len(args) < 2 {
		fmt.Fprintln(os.Stdout, "Please specify an amount and an address")
		return
	}

	amount, err := stringToUint64(args[0])
	if err != nil {
		fmt.Fprintf(os.Stdout, "error converting amount string to an integer: %v\n", err)
		return
	}

	tx, err := c.transactor.CreateStandardTx(amount, args[1])
	if err != nil {
		fmt.Fprintf(os.Stdout, "error creating transaction: %v\n", err)
		return
	}

	buf := new(bytes.Buffer)
	if err := transactions.Marshal(buf, tx); err != nil {
		fmt.Fprintf(os.Stdout, "error encoding transaction: %v\n", err)
	}

	c.eventBroker.Publish(string(topics.Tx), buf)
}

func (c *CLI) sendBidCMD(args []string) {
	if c.transactor == nil {
		fmt.Fprintln(os.Stdout, "Please load a wallet before bidding DUSK")
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

	tx, err := c.transactor.CreateBidTx(amount, lockTime)
	if err != nil {
		fmt.Fprintf(os.Stdout, "error creating transaction: %v\n", err)
		return
	}

	buf := new(bytes.Buffer)
	if err := transactions.Marshal(buf, tx); err != nil {
		fmt.Fprintf(os.Stdout, "error encoding transaction: %v\n", err)
	}

	c.eventBroker.Publish(string(topics.Tx), buf)
}

func (c *CLI) sendStakeCMD(args []string) {
	if c.transactor == nil {
		fmt.Fprintln(os.Stdout, "Please load a wallet before staking DUSK")
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

	tx, err := c.transactor.CreateStakeTx(amount, lockTime)
	if err != nil {
		fmt.Fprintf(os.Stdout, "error creating transaction: %v\n", err)
		return
	}

	buf := new(bytes.Buffer)
	if err := transactions.Marshal(buf, tx); err != nil {
		fmt.Fprintf(os.Stdout, "error encoding transaction: %v\n", err)
	}

	c.eventBroker.Publish(string(topics.Tx), buf)
}
