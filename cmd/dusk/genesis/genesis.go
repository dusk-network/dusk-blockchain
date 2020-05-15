package genesis

import (
	"fmt"

	"github.com/urfave/cli"
)

var genBlock = `
{
  "header": {
    "version": 0,
    "height": 0,
    "timestamp": 1589448723,
    "prev-hash": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
    "seed": "pgiPNB5p5LTW6qNPyXNNCEx4RSReHgFifxRlGP/b6Mch",
    "tx-root": "cxI6oGRjBcXQVuJ7rQbXFf7NPPPO+XpetrJvp/Q4ym8=",
    "certificate": {
      "step-one-batched-sig": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
      "step-two-batched-sig": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
      "step": 0,
      "step-one-committee": 0,
      "step-two-committee": 0
    },
    "hash": "wbopIGfSFHt9z/PNzFS5rRZHegdWC+rA9LJ4bQhiLbQ="
  },
  "transactions": [
    {
      "tx": {
        "inputs": null,
        "outputs": [
          {
            "note": {
              "note_type": 0,
              "pos": 0,
              "nonce": null,
              "r_g": null,
              "pk_r": null,
              "value_commitment": null,
              "transparent_blinding_factor": null,
              "encrypted_blinding_factor": null,
              "transparent_value": 256,
              "encrypted_value": null
            },
            "pk": null,
            "value": 0,
            "blinding_factor": null
          }
        ],
        "fee": null,
        "proof": null,
        "data": null
      },
      "provisioners_addresses": [],
      "bg_pk": {
        "a_g": {
          "y": ""
        },
        "b_g": {
          "y": ""
        }
      }
    }
  ]
}
`

// Action prints a genesis
func Action(c *cli.Context) error {
	fmt.Println(genBlock)
	return nil
}
