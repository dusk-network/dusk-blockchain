package query

import "testing"

func TestTxByTxID(t *testing.T) {

	query := `
		{
		   transactions(txid: "9ea3815bf27a89b0429a1483b9907e7091331bee1882c80b9fb753e6d674de65") {
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
        				"txid": "9ea3815bf27a89b0429a1483b9907e7091331bee1882c80b9fb753e6d674de65",
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
		   transactions(txids: ["9ea3815bf27a89b0429a1483b9907e7091331bee1882c80b9fb753e6d674de65","6ea89ed79c970477fbac038b12bdf72a79a29977c6e2a6b6af23f450abb2f5a0"]) {
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
        				"txid": "9ea3815bf27a89b0429a1483b9907e7091331bee1882c80b9fb753e6d674de65",
        				"txtype": "3"
        			},
					{
 						"blockhash": "9467c5e774eb1b4825d08c0599a0b0815fca5dac16d9690026854ed8d1f229c9",
						"txid": "6ea89ed79c970477fbac038b12bdf72a79a29977c6e2a6b6af23f450abb2f5a0",
						"txtype": "3"
					}
        		]
        	}
        }
	`
	assertQuery(t, query, response)
}
