package cli

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"

	ristretto "github.com/bwesterb/go-ristretto"
	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/initiator"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	"github.com/dusk-network/dusk-blockchain/pkg/core/transactor"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	wallet "github.com/dusk-network/dusk-blockchain/pkg/wallet"
	walletdb "github.com/dusk-network/dusk-blockchain/pkg/wallet/database"
	"github.com/dusk-network/dusk-crypto/mlsag"
	"github.com/dusk-network/dusk-wallet/key"

	"github.com/dusk-network/dusk-blockchain/pkg/wallet/transactions"
)

var testnet = byte(2)

// Make sure we only load one wallet
var walletLoaded = false

func createWalletCMD(args []string, publisher wire.EventBroker, rpcBus *wire.RPCBus) {
	if walletLoaded {
		fmt.Fprintln(os.Stdout, "You have already loaded a wallet. Please re-start the node to load another.")
	}

	if args == nil || len(args) < 1 {
		fmt.Fprintf(os.Stdout, commandInfo["createwallet"]+"\n")
		return
	}
	password := args[0]

	db, err := walletdb.New(cfg.Get().Wallet.Store)
	if err != nil {
		fmt.Fprintf(os.Stdout, "error opening database: %v\n", err)
		return
	}

	w, err := wallet.New(rand.Read, testnet, db, fetchDecoys, fetchInputs, password)
	if err != nil {
		fmt.Fprintf(os.Stdout, "error creating wallet: %v\n", err)
		return
	}
	pubAddr, err := w.PublicAddress()
	if err != nil {
		fmt.Fprintf(os.Stdout, "error attempting to get your public address: %v\n", err)
		return
	}

	fmt.Fprintf(os.Stdout, "Wallet created successfully!\n")
	fmt.Fprintf(os.Stdout, "Public Address: %s\n", pubAddr)

	walletLoaded = true
	transactor.Launch(publisher, w)

	if !cfg.Get().General.WalletOnly {
		initiator.LaunchConsensus(publisher, rpcBus, w)
	}
}

func loadWalletCMD(args []string, publisher wire.EventBroker, rpcBus *wire.RPCBus) {
	if walletLoaded {
		fmt.Fprintln(os.Stdout, "You have already loaded a wallet. Please re-start the node to load another.")
	}

	if args == nil || len(args) < 1 {
		fmt.Fprintf(os.Stdout, commandInfo["loadwallet"]+"\n")
		return
	}
	password := args[0]

	w, err := loadWallet(password)
	if err != nil {
		fmt.Fprintf(os.Stdout, "error attempting to load wallet: %v\n", err)
		return
	}

	pubAddr, err := w.PublicAddress()
	if err != nil {
		fmt.Fprintf(os.Stdout, "error attempting to get your public address: %v\n", err)
		return
	}

	fmt.Fprintf(os.Stdout, "Wallet loaded successfully!\n")
	fmt.Fprintf(os.Stdout, "Public Address: %s\n", pubAddr)

	walletLoaded = true
	transactor.Launch(publisher, w)

	if !cfg.Get().General.WalletOnly {
		initiator.LaunchConsensus(publisher, rpcBus, w)
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

func createFromSeedCMD(args []string, publisher wire.EventBroker, rpcBus *wire.RPCBus) {
	if walletLoaded {
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

	walletLoaded = true
	transactor.Launch(publisher, w)

	if !cfg.Get().General.WalletOnly {
		initiator.LaunchConsensus(publisher, rpcBus, w)
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

func balanceCMD(args []string, eventBroker wire.EventBroker, rpcBus *wire.RPCBus) {
	if !walletLoaded {
		fmt.Fprintln(os.Stdout, "Please load a wallet before checking your balance")
		return
	}

	req := wire.NewRequest(bytes.Buffer{}, 2)
	buf, err := rpcBus.Call(wire.GetBalance, req)
	if err != nil {
		fmt.Fprintf(os.Stdout, "error getting balance: %v\n", err)
		return
	}

	var balance float64
	if err := binary.Read(&buf, binary.LittleEndian, balance); err != nil {
		fmt.Fprintf(os.Stdout, "error reading response: %v\n", err)
		return
	}

	fmt.Fprintln(os.Stdout, balance)
}

func transferCMD(args []string, eventBroker wire.EventBroker, rpcBus *wire.RPCBus) {
	if !walletLoaded {
		fmt.Fprintln(os.Stdout, "Please load a wallet before sending DUSK")
		return
	}

	if len(args) < 2 {
		fmt.Fprintln(os.Stdout, "Please specify an amount and an address")
		return
	}

	buf := new(bytes.Buffer)

	amountInt, err := stringToUint64(args[0])
	if err != nil {
		fmt.Fprintf(os.Stdout, "error converting amount to integer value: %v\n", err)
		return
	}

	if err := encoding.WriteUint64(buf, binary.LittleEndian, amountInt); err != nil {
		fmt.Fprintf(os.Stdout, "%v\n", err)
		return
	}

	if _, err := buf.Write([]byte(args[1])); err != nil {
		fmt.Fprintf(os.Stdout, "%v\n", err)
		return
	}

	eventBroker.Publish(string(topics.Transfer), buf)
}

func sendBidCMD(args []string, eventBroker wire.EventBroker, rpcBus *wire.RPCBus) {
	if !walletLoaded {
		fmt.Fprintln(os.Stdout, "Please load a wallet before bidding DUSK")
		return
	}

	eventBroker.Publish(string(topics.Bid), nil)
}

func sendStakeCMD(args []string, eventBroker wire.EventBroker, rpcBus *wire.RPCBus) {
	if !walletLoaded {
		fmt.Fprintln(os.Stdout, "Please load a wallet before staking DUSK")
		return
	}

	eventBroker.Publish(string(topics.Stake), nil)
}

func fetchDecoys(numMixins int) []mlsag.PubKeys {
	_, db := heavy.CreateDBConnection()

	var pubKeys []mlsag.PubKeys
	var decoys []ristretto.Point
	db.View(func(t database.Transaction) error {
		decoys = t.FetchDecoys(numMixins)
		return nil
	})

	// Potential panic if the database does not have enough decoys
	for i := 0; i < numMixins; i++ {

		var keyVector mlsag.PubKeys
		keyVector.AddPubKey(decoys[i])

		var secondaryKey ristretto.Point
		secondaryKey.Rand()
		keyVector.AddPubKey(secondaryKey)

		pubKeys = append(pubKeys, keyVector)
	}
	return pubKeys
}

func generateDualKey() mlsag.PubKeys {
	pubkeys := mlsag.PubKeys{}

	for i := 0; i < 2; i++ {
		var key ristretto.Point
		key.Rand()
		pubkeys.AddPubKey(key)
	}

	return pubkeys
}

func fetchInputs(netPrefix byte, db *walletdb.DB, totalAmount int64, key *key.Key) ([]*transactions.Input, int64, error) {
	// Fetch all inputs from database that are >= totalAmount
	// returns error if inputs do not add up to total amount
	privSpend, err := key.PrivateSpend()
	if err != nil {
		return nil, 0, err
	}
	return db.FetchInputs(privSpend.Bytes(), totalAmount)
}
