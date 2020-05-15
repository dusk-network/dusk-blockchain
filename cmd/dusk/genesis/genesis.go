package genesis

import (
	"encoding/json"
	"fmt"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/urfave/cli"
)

// Action prints a genesis
func Action(c *cli.Context) error {
	blk := cfg.DecodeGenesis()
	b, err := json.MarshalIndent(blk, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(b))
	return nil
}
