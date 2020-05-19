package query

import (
	"fmt"
	"testing"
)

func TestTxByTxID(t *testing.T) {

	query := fmt.Sprintf(`
		{
		   transactions(txid: "%s") {
			  txid
			  txtype
			  blockhash
		  }
		}
		`, bid1Hash)
	response := fmt.Sprintf(`
		{
			"data": {
				"transactions": [
					{
						"blockhash": "%s",
						"txid": "%s",
						"txtype": "bid"
					}
				]
			}
		}
	`, block1, bid1Hash)
	assertQuery(t, query, response)
}

func TestTxByTxIDs(t *testing.T) {

	query := fmt.Sprintf(`
		{
		   transactions(txids: ["%s", "%s"]) {
			  txid
			  txtype
			  blockhash
		  }
		}
		`, bid3Hash, bid2Hash)
	response := fmt.Sprintf(`
		{
			"data": {
				"transactions": [
					{
						"blockhash": "%s",
						"txid": "%s",
						"txtype": "bid"
					},
					{
						"blockhash": "%s",
						"txid": "%s",
						"txtype": "bid"
					}
				]
			}
		}
	`, block3, bid3Hash, block2, bid2Hash)
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
	response := fmt.Sprintf(`
		{
			"data": {
				"transactions": [
					{
						"txid": "%s",
						"txtype": "bid"
					},
					{
						"txid": "%s",
						"txtype": "bid"
					}
				]
			}
		}
	`, bid3Hash, bid2Hash)
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
	response := fmt.Sprintf(`
	{
		"data": {
			"transactions": [
				{
					"output": [
						{
							"pubkey": "33443344"
						}
					],
					"txid": "%s"
				}
			]
		}
	}
	`, bid3Hash)
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
	response := fmt.Sprintf(`
	{
		"data": {
			"transactions": [
				{
					"input": [
						{
							"keyimage": "5566"
						}
					],
					"txid": "%s"
				}
			]
		}
	}
	`, bid3Hash)
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
						"size": 322,
						"txtype": "bid"
					}
				]
			}
		}
	`
	assertQuery(t, query, response)
}
