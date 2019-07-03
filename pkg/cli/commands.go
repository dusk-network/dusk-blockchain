package cli

import (
	"fmt"

	"gitlab.dusk.network/dusk-core/dusk-wallet/database"
	wallet "gitlab.dusk.network/dusk-core/dusk-wallet/wallet/v3"
)

// CLICommands holds all of the wallet commands that the user can call through
// the interactive shell.
var CLICommands = map[string]func([]string){
	"help":           nil,
	"createwallet":   createWallet,
	"transfer":       transfer,
	"stake":          nil,
	"bid":            nil,
	"startconsensus": nil,
	"exit":           nil,
	"quit":           nil,
}

func createWallet(args []string) {
	// TODO: add proper path
	db, err := database.New("")
	if err != nil {
		// TODO: use logger over fmt.Print
		fmt.Printf("error opening database: %v\n", err)
		return
	}

	// TODO: add functions
	w, err := wallet.New(testnet, db, nil, nil)
	if err != nil {
		fmt.Printf("error creating wallet: %v\n", err)
		return
	}
}

func transfer(args []string) {
	if args == nil {
		fmt.Printf("")
		return
	}

	w, err := wallet.LoadWallet()
	if err != nil {
		fmt.Printf("error opening wallet: %v\n", err)
		return
	}

	// TODO: decide on fee in a proper way
	tx, err := w.NewStandardTx(100)
	if err != nil {
		fmt.Printf("error creating tx: %v\n", err)
	}

}
