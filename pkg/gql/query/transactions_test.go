package query

import "testing"

func TestTxByTxID(t *testing.T) {

	query := `
		{
		   transactions(txid: "4608a001055b8e08203b1230b0e143e3d6444cd63e0c8b4a54535e2fa45dacbf") {
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
						"txid": "4608a001055b8e08203b1230b0e143e3d6444cd63e0c8b4a54535e2fa45dacbf",
						"txtype": "3"
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
		   transactions(txids: ["c9f8cea7df8fdd5c5e1faf264419d19c33aa49767ec0cb18de4df18dba176e29",
		   "5a38c3733b9868a89c05f473c74f909fe38b69f74adca68023075452c1d42f69"]) {
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
						"txid": "c9f8cea7df8fdd5c5e1faf264419d19c33aa49767ec0cb18de4df18dba176e29",
						"txtype": "3"
					},
					{
						"blockhash": "9bf50e394bb81346f8b8db42bddd285ac344260c024a0df808baf7601417d748",
						"txid": "5a38c3733b9868a89c05f473c74f909fe38b69f74adca68023075452c1d42f69",
						"txtype": "3"
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
						"txid": "c9f8cea7df8fdd5c5e1faf264419d19c33aa49767ec0cb18de4df18dba176e29",
						"txtype": "3"
					},
					{
						"txid": "5a38c3733b9868a89c05f473c74f909fe38b69f74adca68023075452c1d42f69",
						"txtype": "3"
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
					"txid": "c9f8cea7df8fdd5c5e1faf264419d19c33aa49767ec0cb18de4df18dba176e29"
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
					"txid": "c9f8cea7df8fdd5c5e1faf264419d19c33aa49767ec0cb18de4df18dba176e29"
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
						"size": 274,
						"txtype": "3"
					}
				]
			}
		}
	`
	assertQuery(t, query, response)
}
