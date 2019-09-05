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
		  blocks(hash: "194dd13ee8a60ac017a82c41c0e2c02498d75f48754351072f392a085d469620") {
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
        					"hash": "194dd13ee8a60ac017a82c41c0e2c02498d75f48754351072f392a085d469620",
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
		  blocks(hashes: ["9bf50e394bb81346f8b8db42bddd285ac344260c024a0df808baf7601417d748", 
                          "9467c5e774eb1b4825d08c0599a0b0815fca5dac16d9690026854ed8d1f229c9"] ) {
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
        					"hash": "9bf50e394bb81346f8b8db42bddd285ac344260c024a0df808baf7601417d748",
        					"height": 1
        				}
        			},
        			{
        				"header": {
        					"hash": "9467c5e774eb1b4825d08c0599a0b0815fca5dac16d9690026854ed8d1f229c9",
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
        						"txid": "6ea89ed79c970477fbac038b12bdf72a79a29977c6e2a6b6af23f450abb2f5a0",
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

func TestLastBlocks(t *testing.T) {
	query := `
		{
		  blocks(last: 3) {
			header {
			   height
				hash
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
        					"hash": "194dd13ee8a60ac017a82c41c0e2c02498d75f48754351072f392a085d469620",
        					"height": 0
        				}
        			},
        			{
        				"header": {
        					"hash": "9bf50e394bb81346f8b8db42bddd285ac344260c024a0df808baf7601417d748",
        					"height": 1
        				}
        			},
        			{
        				"header": {
        					"hash": "9467c5e774eb1b4825d08c0599a0b0815fca5dac16d9690026854ed8d1f229c9",
        					"height": 2
        				}
        			}
        		]
        	}
        }
	`
	assertQuery(t, query, response)
}
