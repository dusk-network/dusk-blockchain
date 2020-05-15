package query

import "testing"

func TestTxByTxID(t *testing.T) {

	query := `
		{
		   transactions(txid: "529086a45bdf1efa27c7291b838018543061cf8aec2632fbbc4856052c4e88aa") {
			  txid
			  txtype
			  blockhash
		  }
		}
		`
	response := `
		{
			"data": {
				"transactions": [
					{
						"blockhash": "194dd13ee8a60ac017a82c41c0e2c02498d75f48754351072f392a085d469620",
						"txid": "529086a45bdf1efa27c7291b838018543061cf8aec2632fbbc4856052c4e88aa",
						"txtype": "bid"
					}
				]
			}
		}
	`
	assertQuery(t, query, response)
}

func TestTxByTxIDs(t *testing.T) {

	query := `
		{
		   transactions(txids: ["1295fbb89ad4a6dcabd7d64651ce7b4f8ca1229c5824fdbb3e5343a52d478817",
		   "9a24a67f4ca71d716e467674293ab33df0629c9e66b47c1cf4df787e250632a8"]) {
			  txid
			  txtype
			  blockhash
		  }
		}
		`
	response := `
		{
			"data": {
				"transactions": [
					{
						"blockhash": "9467c5e774eb1b4825d08c0599a0b0815fca5dac16d9690026854ed8d1f229c9",
						"txid": "1295fbb89ad4a6dcabd7d64651ce7b4f8ca1229c5824fdbb3e5343a52d478817",
						"txtype": "bid"
					},
					{
						"blockhash": "9bf50e394bb81346f8b8db42bddd285ac344260c024a0df808baf7601417d748",
						"txid": "9a24a67f4ca71d716e467674293ab33df0629c9e66b47c1cf4df787e250632a8",
						"txtype": "bid"
					}
				]
			}
		}
	`
	assertQuery(t, query, response)
}

func TestLastTxs(t *testing.T) {

	query := `
		{ 
			transactions(last: 2) 
			{ 
				txid 
				txtype
			}
	  	}
		`
	response := `
		{
			"data": {
				"transactions": [
					{
						"txid": "1295fbb89ad4a6dcabd7d64651ce7b4f8ca1229c5824fdbb3e5343a52d478817",
						"txtype": "bid"
					},
					{
						"txid": "9a24a67f4ca71d716e467674293ab33df0629c9e66b47c1cf4df787e250632a8",
						"txtype": "bid"
					}
				]
			}
		}
	`
	assertQuery(t, query, response)
}
func TestTxOutput(t *testing.T) {

	query := `
		{      
			transactions(last: 1)
			{
				txid  
				output
				{
					pubkey
				}         	 
			}
		}		 
	`
	response := `
	{
		"data": {
			"transactions": [
				{
					"output": [
						{
							"pubkey": "33443344"
						}
					],
					"txid": "1295fbb89ad4a6dcabd7d64651ce7b4f8ca1229c5824fdbb3e5343a52d478817"
				}
			]
		}
	}
	`
	assertQuery(t, query, response)
}

func TestTxInput(t *testing.T) {

	query := `
		{      
			transactions(last: 1)
			{
				txid  
				input
				{
					keyimage
				}         	 
			}
		}		 
	`
	response := `
	{
		"data": {
			"transactions": [
				{
					"input": [
						{
							"keyimage": "5566"
						}
					],
					"txid": "1295fbb89ad4a6dcabd7d64651ce7b4f8ca1229c5824fdbb3e5343a52d478817"
				}
			]
		}
	}
	`
	assertQuery(t, query, response)
}

func TestTxSize(t *testing.T) {

	query := `
		{
			transactions(last: 1)
			{
				size
				txtype
			}
		}
	`

	response := `
		{
			"data": {
				"transactions": [
					{
						"size": 258,
						"txtype": "bid"
					}
				]
			}
		}
	`
	assertQuery(t, query, response)
}
