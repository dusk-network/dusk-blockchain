package cli

import (
	"bytes"
	"fmt"
	"os"

	ristretto "github.com/bwesterb/go-ristretto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/key"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/mlsag"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
	wallet "gitlab.dusk.network/dusk-core/dusk-go/pkg/wallet"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/wallet/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/wallet/transactions"
)

var testnet = byte(2)
var dbName = "walletDb"

func createWallet(args []string, publisher wire.EventPublisher) {
	if args == nil || len(args) < 1 {
		fmt.Fprintf(os.Stdout, commandInfo["createwallet"]+"\n")
		return
	}
	password := args[0]

	db, err := database.New(dbName)
	defer db.Close()
	if err != nil {
		// TODO: use logger over fmt.Print
		fmt.Fprintf(os.Stdout, "error opening database: %v\n", err)
		return
	}

	w, err := wallet.New(testnet, db, fetchDecoys, fetchInputs, password)
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
}

func loadWallet(password string) *wallet.Wallet {

	// First load the database
	db, err := database.New(dbName)
	if err != nil {
		// TODO: use logger over fmt.Print
		fmt.Fprintf(os.Stdout, "error opening database: %v\n", err)
		return nil
	}

	// Then load the wallet
	w, err := wallet.LoadFromFile(testnet, db, fetchDecoys, fetchInputs, password)
	if err != nil {
		fmt.Fprintf(os.Stdout, "error creating wallet: %v\n", err)
		return nil
	}

	return w

}

func transfer(args []string, publisher wire.EventPublisher) {
	if args == nil || len(args) < 4 {
		fmt.Fprintf(os.Stdout, commandInfo["transfer"]+"\n")
		return
	}

	amount, err := stringToScalar(args[0])
	if err != nil {
		fmt.Fprintf(os.Stdout, fmt.Sprintf("%s\n", err.Error()))
	}

	address := args[1]

	fee, err := stringToInt64(args[2])
	if err != nil {
		fmt.Fprintf(os.Stdout, fmt.Sprintf("%s\n", err.Error()))
	}

	password := args[3]

	// Load wallet using password
	w := loadWallet(password)

	// Create a new standard tx
	tx, err := w.NewStandardTx(int64(fee))
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

	publisher.Publish(string(topics.Tx), buf)
}

func sendStake(args []string, publisher wire.EventPublisher) {
	if args == nil || len(args) < 4 {
		fmt.Fprintf(os.Stdout, commandInfo["stake"]+"\n")
		return
	}

	amount, err := stringToScalar(args[0])
	if err != nil {
		fmt.Fprintf(os.Stdout, "%s\n", err.Error())
	}

	lockTime, err := stringToUint64(args[1])
	if err != nil {
		fmt.Fprintf(os.Stdout, "%s\n", err.Error())
	}

	fee, err := stringToInt64(args[2])
	if err != nil {
		fmt.Fprintf(os.Stdout, "%s\n", err.Error())
	}

	password := args[3]

	// Load wallet using password
	w := loadWallet(password)

	// Create a new stake tx
	tx, err := w.NewStakeTx(int64(fee), lockTime, amount)
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

	publisher.Publish(string(topics.Tx), buf)
}

func sendBid(args []string, publisher wire.EventPublisher) {
	if args == nil || len(args) < 4 {
		fmt.Fprintf(os.Stdout, commandInfo["bid"]+"\n")
		return
	}

	amount, err := stringToScalar(args[0])
	if err != nil {
		fmt.Fprintf(os.Stdout, "%s\n", err.Error())
	}

	lockTime, err := stringToUint64(args[1])
	if err != nil {
		fmt.Fprintf(os.Stdout, "%s\n", err.Error())
	}

	fee, err := stringToInt64(args[2])
	if err != nil {
		fmt.Fprintf(os.Stdout, "%s\n", err.Error())
	}

	password := args[3]

	// Load wallet using password
	w := loadWallet(password)

	// Create a new bid tx
	tx, err := w.NewBidTx(int64(fee), lockTime, amount)
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

	publisher.Publish(string(topics.Tx), buf)
}

func fetchDecoys(numMixins int) []mlsag.PubKeys {
	var pubKeys []mlsag.PubKeys
	for i := 0; i < numMixins; i++ {
		pubKeyVector := generateDualKey()
		pubKeys = append(pubKeys, pubKeyVector)
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

func fetchInputs(netPrefix byte, db *database.DB, totalAmount int64, key *key.Key) ([]*transactions.Input, int64, error) {
	// Fetch all inputs from database that are >= totalAmount
	// returns error if inputs do not add up to total amount
	privSpend, err := key.PrivateSpend()
	if err != nil {
		return nil, 0, err
	}
	return db.FetchInputs(privSpend.Bytes(), totalAmount)
}
