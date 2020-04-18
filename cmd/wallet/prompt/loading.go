package prompt

import (
	"context"
	"errors"

	"github.com/dusk-network/dusk-protobuf/autogen/go/node"
	"github.com/manifoldco/promptui"
)

func loadWallet(client node.NodeClient) (*node.LoadResponse, error) {
	pw := getPassword()
	return client.LoadWallet(context.Background(), &node.LoadRequest{Password: pw})
}

func createWallet(client node.NodeClient) (*node.LoadResponse, error) {
	pw := getPassword()
	return client.CreateWallet(context.Background(), &node.CreateRequest{Password: pw})
}

func loadFromSeed(client node.NodeClient) (*node.LoadResponse, error) {
	validate := func(input string) error {
		if len(input) < 64 {
			return errors.New("Seed must be 64 characters or more")
		}

		return nil
	}

	prompt := promptui.Prompt{
		Label:    "Seed",
		Validate: validate,
		Mask:     '*',
	}

	seed, err := prompt.Run()
	if err != nil {
		panic(err)
	}

	pw := getPassword()
	return client.CreateFromSeed(context.Background(), &node.CreateRequest{Password: pw, Seed: []byte(seed)})
}

func getPassword() string {
	prompt := promptui.Prompt{
		Label: "Password",
		Mask:  '*',
	}

	pw, err := prompt.Run()
	if err != nil {
		panic(err)
	}

	return pw
}
