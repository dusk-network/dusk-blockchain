// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package prompt

import (
	"context"
	"strconv"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/wallet"
	"github.com/dusk-network/dusk-protobuf/autogen/go/node"
	"github.com/manifoldco/promptui"
)

func transferDusk(client node.TransactorClient) (*node.TransactionResponse, error) {
	amount := getAmount()

	// FIXME: 493 - there should be syntax-validation of address
	validateAddress := func(input string) error {
		// FIXME: there is no ToKey method in the wallet currently
		//address := key.PublicAddress(input)
		//// TODO: use netprefix inferred from config
		//if _, err := address.ToKey(2); err != nil {
		//	return err
		//}
		return nil
	}

	addressPrompt := promptui.Prompt{
		Label:    "Address",
		Validate: validateAddress,
	}

	address, err := addressPrompt.Run()
	if err != nil {
		return nil, err
	}

	return client.Transfer(context.Background(), &node.TransferRequest{Amount: amount, Address: []byte(address)})
}

func stakeDusk(client node.TransactorClient) (*node.TransactionResponse, error) {
	amount := getAmount()
	lockTime := getLockTime()
	// TODO: parameterize fee
	return client.Stake(context.Background(), &node.StakeRequest{Amount: amount, Fee: 100, Locktime: lockTime})
}

func getAmount() uint64 {
	validate := func(input string) error {
		if _, err := strconv.ParseFloat(input, 64); err != nil {
			return err
		}

		return nil
	}

	prompt := promptui.Prompt{
		Label:    "Amount",
		Validate: validate,
	}

	amountString, err := prompt.Run()
	if err != nil {
		panic(err)
	}

	amountFloat, _ := strconv.ParseFloat(amountString, 64)
	amount := uint64(amountFloat * float64(wallet.DUSK))
	return amount
}

func getLockTime() uint64 {
	validate := func(input string) error {
		if _, err := strconv.Atoi(input); err != nil {
			return err
		}

		return nil
	}

	prompt := promptui.Prompt{
		Label:    "Lock Time",
		Validate: validate,
	}

	lockTimeString, err := prompt.Run()
	if err != nil {
		panic(err)
	}

	lockTime, _ := strconv.Atoi(lockTimeString)
	return uint64(lockTime)
}
