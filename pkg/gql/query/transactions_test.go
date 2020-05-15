package query

import "testing"

func TestTxByTxID(t *testing.T) {

	query := `
		{
		   transactions(txid: "2451bad48209f51fd619e964724beafbd024aeacbf16a712d18fdf211d85b0ac") {
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
						"txid": "2451bad48209f51fd619e964724beafbd024aeacbf16a712d18fdf211d85b0ac",
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
		   transactions(txids: ["a12ae441edf18f6c4f8a38492bdd6905e07a8cd4e29b2a2ccf42b30234608dcf",
		   "1ec314e631b585450045d1a210c83c2b1826130259588f2fb690ef9386dd85ce"]) {
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
						"txid": "a12ae441edf18f6c4f8a38492bdd6905e07a8cd4e29b2a2ccf42b30234608dcf",
						"txtype": "bid"
					},
					{
						"blockhash": "9bf50e394bb81346f8b8db42bddd285ac344260c024a0df808baf7601417d748",
						"txid": "1ec314e631b585450045d1a210c83c2b1826130259588f2fb690ef9386dd85ce",
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
						"txid": "a12ae441edf18f6c4f8a38492bdd6905e07a8cd4e29b2a2ccf42b30234608dcf",
						"txtype": "bid"
					},
					{
						"txid": "1ec314e631b585450045d1a210c83c2b1826130259588f2fb690ef9386dd85ce",
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
					"txid": "a12ae441edf18f6c4f8a38492bdd6905e07a8cd4e29b2a2ccf42b30234608dcf"
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
					"txid": "a12ae441edf18f6c4f8a38492bdd6905e07a8cd4e29b2a2ccf42b30234608dcf"
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
						"size": 275,
						"txtype": "bid"
					}
				]
			}
		}
	`
	assertQuery(t, query, response)
}
