// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

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
			  gaslimit
			  gasprice
			  feepaid
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
						"txtype": "3",
						"gaslimit": 50000,
						"gasprice": 100,
						"feepaid": 5000000
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
						"txtype": "3"
					},
					{
						"blockhash": "%s",
						"txid": "%s",
						"txtype": "3"
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
						"txtype": "3"
					},
					{
						"txid": "%s",
						"txtype": "3"
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
							"pubkey": "0000000000000000000000000000000000000000000000000000000000000000"
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
							"keyimage": "0000000000000000000000000000000000000000000000000000000000000000"
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
						"size": 713,
						"txtype": "3"
					}
				]
			}
		}
	`
	assertQuery(t, query, response)
}
