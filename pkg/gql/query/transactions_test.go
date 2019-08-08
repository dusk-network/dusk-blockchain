package query

import "testing"

func TestTxByTxID(t *testing.T) {

	// TODO: Depends on resolveTx issue
	t.SkipNow()

	query := `
		{
		   transactions(txid: "yiTTM/cKZT5M8/ccjAHiJRap3RAa9wQq66Z7VxFSGLw=") {
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
        				"blockhash": "m/UOOUu4E0b4uNtCvd0oWsNEJgwCSg34CLr3YBQX10g=",
        				"txid": "yiTTM/cKZT5M8/ccjAHiJRap3RAa9wQq66Z7VxFSGLw=",
        				"txtype": "3"
        			}
        		]
        	}
        }
	`
	assertQuery(t, query, response)
}

func TestTxByTxIDs(t *testing.T) {

	// TODO: Depends on resolveTx issue
	t.SkipNow()

	query := `
		{
		   transactions(txids: ["yiTTM/cKZT5M8/ccjAHiJRap3RAa9wQq66Z7VxFSGLw=","Ts4G1WI0BRutCsW5m5UaAqWdVyTrib3bhVHz1q47JnU="]) {
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
        				"blockhash": "",
        				"txid": "yiTTM/cKZT5M8/ccjAHiJRap3RAa9wQq66Z7VxFSGLw=",
        				"txtype": "3"
        			},
					{
						"txid": "Ts4G1WI0BRutCsW5m5UaAqWdVyTrib3bhVHz1q47JnU=",
						"txtype": "3"
					}
        		]
        	}
        }
	`
	assertQuery(t, query, response)
}
