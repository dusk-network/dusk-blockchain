package query

import "testing"

func TestTxByTxID(t *testing.T) {

	query := `
		{
		   transactions(txid: "05300b8d9904a31241520bb2961ef2516401884b8f6fc3862542b13baa4089cc") {
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
						"txid": "05300b8d9904a31241520bb2961ef2516401884b8f6fc3862542b13baa4089cc",
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
		   transactions(txids: ["6adef894526715190947eee09832bc1cb5b21880a03c0518f2f52c42db77f955",
		   "c1ecbbaab214ae2dc230c5adf57b0a13349cb1d1eb8fdfbc7722a5baa6276de8"]) {
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
						"txid": "6adef894526715190947eee09832bc1cb5b21880a03c0518f2f52c42db77f955",
						"txtype": "3"
					},
					{
						"blockhash": "9bf50e394bb81346f8b8db42bddd285ac344260c024a0df808baf7601417d748",
						"txid": "c1ecbbaab214ae2dc230c5adf57b0a13349cb1d1eb8fdfbc7722a5baa6276de8",
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
						"txid": "6adef894526715190947eee09832bc1cb5b21880a03c0518f2f52c42db77f955",
						"txtype": "3"
					},
					{
						"txid": "c1ecbbaab214ae2dc230c5adf57b0a13349cb1d1eb8fdfbc7722a5baa6276de8",
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
							"pubkey": "ea2c58c43d2ac9783a25dae2399b227fc1fd2a8bca41ca34aef74c9a3f7b435f"
						}
					],
					"txid": "6adef894526715190947eee09832bc1cb5b21880a03c0518f2f52c42db77f955"
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
							"keyimage": "d886641e16a1165d70fa89413c4129d56b15d5f44d2dd2b09823cd723487656a"
						}
					],
					"txid": "6adef894526715190947eee09832bc1cb5b21880a03c0518f2f52c42db77f955"
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
						"size": 766,
						"txtype": "3"
					}
				]
			}
		}
	`
	assertQuery(t, query, response)
}
