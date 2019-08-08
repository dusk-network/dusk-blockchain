package query

import (
	"testing"
)

func TestBlocksByHeight(t *testing.T) {
	query := `
		{
		  tip: blocks(height: -1) {
			header {
			   height
			}
		  },
		  genesis: blocks(height: 0) {
			header {
			   height
			}
		  }
		}
		`
	response := `
		{  
		  "data":{  
			"genesis":[
			  {  
				"header":{  
				  "height":0
				}
			  }
			],
			"tip":[  
			  {  
				"header":{  
				  "height":2
				}
			  }
			]
		  }
		}
	`
	assertQuery(t, query, response)
}

func TestBlocksByHash(t *testing.T) {
	query := `
		{
		  blocks(hash: "GU3RPuimCsAXqCxBwOLAJJjXX0h1Q1EHLzkqCF1GliA=") {
			header {
               hash
			   height
			}
		  },
		}
		`
	response := `
		{
        	"data": {
        		"blocks": [
        			{
        				"header": {
        					"hash": "GU3RPuimCsAXqCxBwOLAJJjXX0h1Q1EHLzkqCF1GliA=",
        					"height": 0
        				}
        			}
        		]
        	}
        }
	`
	assertQuery(t, query, response)
}

func TestBlocksByHashes(t *testing.T) {
	query := `
		{
		  blocks(hashes: ["m/UOOUu4E0b4uNtCvd0oWsNEJgwCSg34CLr3YBQX10g=", "lGfF53TrG0gl0IwFmaCwgV/KXawW2WkAJoVO2NHyKck="] ) {
			header {
               hash
			   height
			}
		  },
		}
		`
	response := `
		{
        	"data": {
        		"blocks": [
        			{
        				"header": {
        					"hash": "m/UOOUu4E0b4uNtCvd0oWsNEJgwCSg34CLr3YBQX10g=",
        					"height": 1
        				}
        			},
        			{
        				"header": {
        					"hash": "lGfF53TrG0gl0IwFmaCwgV/KXawW2WkAJoVO2NHyKck=",
        					"height": 2
        				}
        			}
        		]
        	}
        }
	`
	assertQuery(t, query, response)

	// Test Blocks By Range (same response expected)
	query = `
		{
		  blocks(range: [1,2] ) {
			header {
               hash
			   height
			}
		  },
		}
		`
	assertQuery(t, query, response)
}

func TestBlocksTxs(t *testing.T) {
	query := `
		{
		  blocks(height: -1) {
			header {
			   height
			}
			transactions {
				txid
				txtype
			}
		  }
		}
		`
	response := `
		{
        	"data": {
        		"blocks": [
        			{
        				"header": {
        					"height": 2
        				},
        				"transactions": [
        					{
        						"txid": "Ts4G1WI0BRutCsW5m5UaAqWdVyTrib3bhVHz1q47JnU=",
        						"txtype": "3"
        					}
        				]
        			}
        		]
        	}
        }
	`
	assertQuery(t, query, response)
}
