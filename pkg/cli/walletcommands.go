package cli

import (
	"bytes"
	"fmt"
	"os"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/key"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
	wallet "gitlab.dusk.network/dusk-core/dusk-go/pkg/wallet"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/wallet/database"
)

var dbName = "walletDb"

func createWallet(args []string, publisher wire.EventPublisher) {

	password := args[1]

	// TODO: add proper path
	db, err := database.New(dbName)
	defer db.Close()
	if err != nil {
		// TODO: use logger over fmt.Print
		fmt.Fprintf(os.Stdout, "error opening database: %v\n", err)
		return
	}

	// TODO: add functions
	_, err = wallet.New(2, db, nil, nil, password)
	if err != nil {
		fmt.Fprintf(os.Stdout, "error creating wallet: %v\n", err)
		return
	}
}

func loadWallet(password string) *wallet.Wallet {

	// First load the database
	db, err := database.New(dbName)
	defer db.Close()
	if err != nil {
		// TODO: use logger over fmt.Print
		fmt.Fprintf(os.Stdout, "error opening database: %v\n", err)
		return nil
	}

	// Then load the wallet
	// TODO: add functions
	w, err := wallet.Load(2, db, nil, nil, password)
	if err != nil {
		fmt.Fprintf(os.Stdout, "error creating wallet: %v\n", err)
		return nil
	}

	return w

}

func transfer(args []string, publisher wire.EventPublisher) {
	if args == nil || len(args) < 5 {
		fmt.Fprintf(os.Stdout, commandInfo["transfer"])
		return
	}

	amount, err := stringToScalar(args[1])
	if err != nil {
		fmt.Fprintf(os.Stdout, fmt.Sprintf("%s\n", err.Error()))
	}

	address := args[2]

	fee, err := stringToInt64(args[3])
	if err != nil {
		fmt.Fprintf(os.Stdout, fmt.Sprintf("%s\n", err.Error()))
	}

	password := args[4]

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

	// Convert wallet-tx to wireTx and encode into buffer
	wireTx, err := tx.WireStandardTx()
	if err != nil {
		fmt.Fprintf(os.Stdout, fmt.Sprintf("%s\n", err.Error()))
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
	if args == nil || len(args) < 5 {
		fmt.Fprintf(os.Stdout, commandInfo["stake"])
		return
	}

	amount, err := stringToScalar(args[1])
	if err != nil {
		fmt.Fprintf(os.Stdout, fmt.Sprintf("%s\n", err.Error()))
	}

	lockTime, err := stringToUint64(args[2])
	if err != nil {
		fmt.Fprintf(os.Stdout, fmt.Sprintf("%s\n", err.Error()))
	}

	fee, err := stringToInt64(args[3])
	if err != nil {
		fmt.Fprintf(os.Stdout, fmt.Sprintf("%s\n", err.Error()))
	}

	password := args[4]

	// Load wallet using password
	w := loadWallet(password)

	// Create a new stake tx
	tx, err := w.NewStakeTx(int64(fee), lockTime, amount)
	if err != nil {
		fmt.Fprintf(os.Stdout, "error creating tx: %v\n", err)
		return
	}

	// Convert wallet-tx to wireTx and encode into buffer
	wireTx, err := tx.WireStake()
	if err != nil {
		fmt.Fprintf(os.Stdout, fmt.Sprintf("%s\n", err.Error()))
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
	if args == nil || len(args) < 5 {
		fmt.Fprintf(os.Stdout, commandInfo["bid"])
		return
	}

	amount, err := stringToScalar(args[1])
	if err != nil {
		fmt.Fprintf(os.Stdout, fmt.Sprintf("%s\n", err.Error()))
	}

	lockTime, err := stringToUint64(args[2])
	if err != nil {
		fmt.Fprintf(os.Stdout, fmt.Sprintf("%s\n", err.Error()))
	}

	fee, err := stringToInt64(args[3])
	if err != nil {
		fmt.Fprintf(os.Stdout, fmt.Sprintf("%s\n", err.Error()))
	}

	password := args[4]

	// Load wallet using password
	w := loadWallet(password)

	// Create a new bid tx
	tx, err := w.NewBidTx(int64(fee), lockTime, amount)
	if err != nil {
		fmt.Fprintf(os.Stdout, "error creating tx: %v\n", err)
		return
	}

	// Convert wallet-tx to wireTx and encode into buffer
	wireTx, err := tx.WireBid()
	if err != nil {
		fmt.Fprintf(os.Stdout, fmt.Sprintf("%s\n", err.Error()))
		return
	}
	buf := new(bytes.Buffer)
	if err := wireTx.Encode(buf); err != nil {
		fmt.Fprintf(os.Stdout, "error encoding tx: %v\n", err)
		return
	}

	publisher.Publish(string(topics.Tx), buf)
}
