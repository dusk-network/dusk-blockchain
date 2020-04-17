package prompt

import (
	"context"
	"strconv"

	"github.com/dusk-network/dusk-protobuf/autogen/go/node"
	"github.com/dusk-network/dusk-wallet/v2/key"
	"github.com/dusk-network/dusk-wallet/v2/wallet"
	"github.com/manifoldco/promptui"
)

func transferDusk(client node.NodeClient) (*node.TransferResponse, error) {
	amount := getAmount()

	validateAddress := func(input string) error {
		address := key.PublicAddress(input)
		// TODO: use netprefix inferred from config
		if _, err := address.ToKey(2); err != nil {
			return err
		}

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

func bidDusk(client node.NodeClient) (*node.TransferResponse, error) {
	amount := getAmount()
	lockTime := getLockTime()
	return client.SendBid(context.Background(), &node.ConsensusTxRequest{Amount: amount, LockTime: lockTime})
}

func stakeDusk(client node.NodeClient) (*node.TransferResponse, error) {
	amount := getAmount()
	lockTime := getLockTime()
	return client.SendStake(context.Background(), &node.ConsensusTxRequest{Amount: amount, LockTime: lockTime})
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
