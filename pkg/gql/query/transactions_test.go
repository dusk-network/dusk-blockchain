package query

import "testing"

func TestTxByTxID(t *testing.T) {

	query := `
		{
		   transactions(txid: "ca24d333f70a653e4cf3f71c8c01e22516a9dd101af7042aeba67b57115218bc") {
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
        				"blockhash": "9bf50e394bb81346f8b8db42bddd285ac344260c024a0df808baf7601417d748",
        				"txid": "ca24d333f70a653e4cf3f71c8c01e22516a9dd101af7042aeba67b57115218bc",
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
		   transactions(txids: ["ca24d333f70a653e4cf3f71c8c01e22516a9dd101af7042aeba67b57115218bc","4ece06d56234051bad0ac5b99b951a02a59d5724eb89bddb8551f3d6ae3b2675"]) {
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
        				"blockhash": "9bf50e394bb81346f8b8db42bddd285ac344260c024a0df808baf7601417d748",
        				"txid": "ca24d333f70a653e4cf3f71c8c01e22516a9dd101af7042aeba67b57115218bc",
        				"txtype": "3"
        			},
					{
 						"blockhash": "9467c5e774eb1b4825d08c0599a0b0815fca5dac16d9690026854ed8d1f229c9",
						"txid": "4ece06d56234051bad0ac5b99b951a02a59d5724eb89bddb8551f3d6ae3b2675",
						"txtype": "3"
					}
        		]
        	}
        }
	`
	assertQuery(t, query, response)
}
