package cli

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/config"
	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/factory"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/generation"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/msg"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	"github.com/dusk-network/dusk-blockchain/pkg/crypto/key"
	"github.com/dusk-network/dusk-blockchain/pkg/crypto/mlsag"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	wallet "github.com/dusk-network/dusk-blockchain/pkg/wallet"
	walletdb "github.com/dusk-network/dusk-blockchain/pkg/wallet/database"

	"github.com/dusk-network/dusk-blockchain/pkg/wallet/transactions"
)

var testnet = byte(2)

// cliWallet will be used to scan blocks in the background
// when we received a topic.AcceptedBlock
var cliWallet *wallet.Wallet

// DBInstance will be used to close any open connections to
// the database
var DBInstance *walletdb.DB

func createWalletCMD(args []string, publisher wire.EventBroker, rpcBus *wire.RPCBus) {

	if DBInstance != nil {
		DBInstance.Close()
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

	cliWallet = w
	DBInstance = db

	if !cfg.Get().General.WalletOnly {
		go startProvisioner(publisher, rpcBus)
		go startBlockGenerator(publisher, rpcBus)
	}
}

func loadWalletCMD(args []string, publisher wire.EventBroker, rpcBus *wire.RPCBus) {
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

	if !cfg.Get().General.WalletOnly {
		go startProvisioner(publisher, rpcBus)
		go startBlockGenerator(publisher, rpcBus)
	}
}

func loadWallet(password string) (*wallet.Wallet, error) {
	if DBInstance != nil {
		DBInstance.Close()
	}

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

	cliWallet = w
	DBInstance = db

	return w, nil
}

func transferCMD(args []string, publisher wire.EventBroker, rpcBus *wire.RPCBus) {
	if args == nil || len(args) < 3 {
		fmt.Fprintf(os.Stdout, commandInfo["transfer"]+"\n")
		return
	}

	amount, err := stringToScalar(args[0])
	if err != nil {
		fmt.Fprintf(os.Stdout, fmt.Sprintf("%s\n", err.Error()))
		return
	}

	address := args[1]
	password := args[2]

	// Load wallet using password
	w, err := loadWallet(password)
	if err != nil {
		fmt.Fprintf(os.Stdout, "error attempting to load wallet: %v\n", err)
		return
	}

	// Sync wallet
	if err := syncWallet(); err != nil {
		fmt.Fprintf(os.Stdout, "%v", err)
		return
	}

	// Create a new standard tx
	tx, err := w.NewStandardTx(cfg.MinFee)
	if err != nil {
		fmt.Fprintf(os.Stdout, "error creating tx: %v\n", err)
		return
	}

	// Send amount to address
	tx.AddOutput(key.PublicAddress(address), amount)

	// Sign tx
	err = w.Sign(tx)
	if err != nil {
		fmt.Fprintf(os.Stdout, "%s\n", err.Error())
		return
	}

	// Convert wallet-tx to wireTx and encode into buffer
	wireTx, err := tx.WireStandardTx()
	if err != nil {
		fmt.Fprintf(os.Stdout, "%s\n", err.Error())
		return
	}
	buf := new(bytes.Buffer)
	if err := wireTx.Encode(buf); err != nil {
		fmt.Fprintf(os.Stdout, "error encoding tx: %v\n", err)
		return
	}

	_, err = wireTx.CalculateHash()
	if err != nil {
		fmt.Fprintf(os.Stdout, "%s\n", err.Error())
		return
	}
	fmt.Fprintf(os.Stdout, "hash: %s\n", hex.EncodeToString(wireTx.TxID))

	publisher.Publish(string(topics.Tx), buf)
}

func createFromSeedCMD(args []string, publisher wire.EventBroker, rpcBus *wire.RPCBus) {
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

	if !cfg.Get().General.WalletOnly {
		go startProvisioner(publisher, rpcBus)
		go startBlockGenerator(publisher, rpcBus)
	}
}

func createFromSeed(seedBytes []byte, password string) (*wallet.Wallet, error) {

	if DBInstance != nil {
		DBInstance.Close()
	}

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

	cliWallet = w
	DBInstance = db

	return w, nil

}

func sendStakeCMD(args []string, publisher wire.EventBroker, rpcBus *wire.RPCBus) {
	if args == nil || len(args) < 3 {
		fmt.Fprintf(os.Stdout, commandInfo["stake"]+"\n")
		return
	}

	amount, err := stringToScalar(args[0])
	if err != nil {
		fmt.Fprintf(os.Stdout, "%s\n", err.Error())
		return
	}

	lockTime, err := stringToUint64(args[1])
	if err != nil {
		fmt.Fprintf(os.Stdout, "%s\n", err.Error())
		return
	}

	password := args[2]

	// Load wallet using password
	w, err := loadWallet(password)
	if err != nil {
		fmt.Fprintf(os.Stdout, "error attempting to load wallet: %v\n", err)
		return
	}

	// Sync wallet
	if err := syncWallet(); err != nil {
		fmt.Fprintf(os.Stdout, "%v", err)
		return
	}

	// Create a new stake tx
	tx, err := w.NewStakeTx(cfg.MinFee, lockTime, amount)
	if err != nil {
		fmt.Fprintf(os.Stdout, "error creating tx: %v\n", err)
		return
	}

	// Sign tx
	err = w.Sign(tx)
	if err != nil {
		fmt.Fprintf(os.Stdout, "%s\n", err.Error())
		return
	}

	// Convert wallet-tx to wireTx and encode into buffer
	wireTx, err := tx.WireStakeTx()
	if err != nil {
		fmt.Fprintf(os.Stdout, "%s\n", err.Error())
		return
	}
	buf := new(bytes.Buffer)
	if err := wireTx.Encode(buf); err != nil {
		fmt.Fprintf(os.Stdout, "error encoding tx: %v\n", err)
		return
	}

	_, err = wireTx.CalculateHash()
	if err != nil {
		fmt.Fprintf(os.Stdout, "%s\n", err.Error())
		return
	}
	fmt.Fprintf(os.Stdout, "hash: %s\n", hex.EncodeToString(wireTx.TxID))

	publisher.Publish(string(topics.Tx), buf)
}

func sendBidCMD(args []string, publisher wire.EventBroker, rpcBus *wire.RPCBus) {
	if args == nil || len(args) < 3 {
		fmt.Fprintf(os.Stdout, commandInfo["bid"]+"\n")
		return
	}

	amount, err := stringToScalar(args[0])
	if err != nil {
		fmt.Fprintf(os.Stdout, "%s\n", err.Error())
		return
	}

	lockTime, err := stringToUint64(args[1])
	if err != nil {
		fmt.Fprintf(os.Stdout, "%s\n", err.Error())
		return
	}

	password := args[2]

	// Load wallet using password
	w, err := loadWallet(password)
	if err != nil {
		fmt.Fprintf(os.Stdout, "error attempting to load wallet: %v\n", err)
		return
	}

	// Sync wallet
	if err := syncWallet(); err != nil {
		fmt.Fprintf(os.Stdout, "%v", err)
		return
	}

	// Create a new bid tx
	tx, err := w.NewBidTx(cfg.MinFee, lockTime, amount)
	if err != nil {
		fmt.Fprintf(os.Stdout, "error creating tx: %v\n", err)
		return
	}

	// Sign tx
	err = w.Sign(tx)
	if err != nil {
		fmt.Fprintf(os.Stdout, "%s\n", err.Error())
		return
	}

	// Convert wallet-tx to wireTx and encode into buffer
	wireTx, err := tx.WireBid()
	if err != nil {
		fmt.Fprintf(os.Stdout, "%s\n", err.Error())
		return
	}
	buf := new(bytes.Buffer)
	if err := wireTx.Encode(buf); err != nil {
		fmt.Fprintf(os.Stdout, "error encoding tx: %v\n", err)
		return
	}

	_, err = wireTx.CalculateHash()
	if err != nil {
		fmt.Fprintf(os.Stdout, "%s\n", err.Error())
		return
	}
	fmt.Fprintf(os.Stdout, "hash: %s\n", hex.EncodeToString(wireTx.TxID))

	publisher.Publish(string(topics.Tx), buf)
}

func syncWallet() error {
	var totalSpent, totalReceived uint64
	// keep looping until tipHash = currentBlockHash
	for {
		// Get Wallet height
		walletHeight, err := cliWallet.GetSavedHeight()
		if err != nil {
			cliWallet.UpdateWalletHeight(0)
		}

		// Get next block using walletHeight and tipHash of the node
		blk, tipHash, tipHeight, err := fetchBlockHeightAndState(walletHeight)
		if err != nil {
			return fmt.Errorf("\nerror fetching block from node db: %v", err)
		}

		fmt.Fprintf(os.Stdout, "\rSyncing wallet... (%v/%v)", blk.Header.Height, tipHeight)
		// call wallet.CheckBlock
		spentCount, receivedCount, err := cliWallet.CheckWireBlock(*blk)
		if err != nil {
			return fmt.Errorf("\nerror fetching block: %v", err)
		}

		totalSpent += spentCount
		totalReceived += receivedCount

		// check if state is equal to the block that we fetched
		if bytes.Equal(tipHash, blk.Header.Hash) {
			break
		}
	}

	fmt.Fprintf(os.Stdout, "\nFound %d spends and %d receives\n", totalSpent, totalReceived)
	return nil
}

func balanceCMD(args []string, publisher wire.EventBroker, rpcBus *wire.RPCBus) {
	if cliWallet == nil {
		fmt.Fprintf(os.Stdout, "please load a wallet before trying to check balance\n")
		return
	}

	if err := syncWallet(); err != nil {
		fmt.Fprintf(os.Stdout, "%v", err)
		return
	}

	balance, err := cliWallet.Balance()
	if err != nil {
		fmt.Fprintf(os.Stdout, "error fetching balance: %v\n", err)
		return
	}
	fmt.Fprintf(os.Stdout, "Balance: %.8f\n", balance)
}

func startProvisioner(publisher wire.EventBroker, rpcBus *wire.RPCBus) {
	// Setting up the consensus factory
	f := factory.New(publisher, rpcBus, config.ConsensusTimeOut, cliWallet.ConsensusKeys())
	f.StartConsensus()

	blsPubKey := cliWallet.ConsensusKeys().BLSPubKeyBytes

	startingRound := getStartingRound(blsPubKey, publisher)

	// Notify consensus components
	roundBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(roundBytes, startingRound)
	publisher.Publish(msg.InitializationTopic, bytes.NewBuffer(roundBytes))
}

func startBlockGenerator(publisher wire.EventBroker, rpcBus *wire.RPCBus) {
	// make some random keys to sign the seed with
	keys, err := user.NewRandKeys()
	if err != nil {
		fmt.Fprintf(os.Stdout, "error starting block generation component - could not generate keys: %v\n", err)
		return
	}

	// reconstruct k
	zeroPadding := make([]byte, 4)
	privSpend, err := cliWallet.PrivateSpend()
	if err != nil {
		fmt.Fprintf(os.Stdout, "error starting block generation component - could not get private spend: %v\n", err)
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
}

func fetchBlockHeightAndState(height uint64) (*block.Block, []byte, uint64, error) {
	_, db := heavy.CreateDBConnection()

	var blk *block.Block
	var state *database.State
	var tipHeight uint64
	err := db.View(func(t database.Transaction) error {
		hash, err := t.FetchBlockHashByHeight(height)
		if err != nil {
			return err
		}
		state, err = t.FetchState()
		if err != nil {
			return err
		}

		blk, err = t.FetchBlock(hash)
		if err != nil {
			return err
		}

		tipHeight, err = t.FetchCurrentHeight()
		return err
	})
	if err != nil {
		return nil, nil, 0, err
	}

	return blk, state.TipHash, tipHeight, nil
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

	var primaryKey ristretto.Point
	primaryKey.Rand()
	pubkeys.AddPubKey(primaryKey)

	var secondaryKey ristretto.Point
	secondaryKey.Rand()
	pubkeys.AddPubKey(secondaryKey)

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
