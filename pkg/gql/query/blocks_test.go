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

func TestBlocksByHeight(t *testing.T) {
	query := `
		{
		  tip: blocks(height: -1) {
			header {
			   height
			   reward
			   gaslimit
			   feespaid
			}
		  },
		  genesis: blocks(height: 0) {
			header {
			   height
			   reward
			   gaslimit
			   feespaid
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
				  "height":0,
				  "feespaid":0,
				  "gaslimit":0,
				  "reward":0
				}
			  }
			],
			"tip":[  
			  {  
				"header":{  
				  "height":2,
				  "feespaid":0,
				  "gaslimit":0,
				  "reward":16000000000
				}
			  }
			]
		  }
		}
	`
	assertQuery(t, query, response)
}

func TestBlocksByHash(t *testing.T) {
	query := fmt.Sprintf(`
		{
		  blocks(hash: "%s") {
			header {
               hash
			   height
			}
		  },
		}
		`, block1)
	response := fmt.Sprintf(`
		{
        	"data": {
        		"blocks": [
        			{
        				"header": {
        					"hash": "%s",
        					"height": 0
        				}
        			}
        		]
        	}
        }
	`, block1)
	assertQuery(t, query, response)
}

func TestBlocksByHashes(t *testing.T) {
	query := fmt.Sprintf(`
		{
		  blocks(hashes: ["%s", 
                          "%s"] ) {
			header {
               hash
			   height
			}
		  },
		}
		`, block2, block3)
	response := fmt.Sprintf(`
		{
        	"data": {
        		"blocks": [
        			{
        				"header": {
        					"hash": "%s",
        					"height": 1
        				}
        			},
        			{
        				"header": {
        					"hash": "%s",
        					"height": 2
        				}
        			}
        		]
        	}
        }
	`, block2, block3)
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
	t.SkipNow()

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
	response := fmt.Sprintf(`
		{
			"data": {
				"blocks": [
					{
						"header": {
							"height": 2
						},
						"transactions": [
							{
								"txid": "%s",
								"txtype": "3"
							}
						]
					}
				]
			}
		}
	`, bid3Hash)
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
	response := fmt.Sprintf(`
		 {
        	"data": {
        		"blocks": [
        			{
        				"header": {
        					"hash": "%s",
        					"height": 0
        				}
        			},
        			{
        				"header": {
        					"hash": "%s",
        					"height": 1
        				}
        			},
        			{
        				"header": {
        					"hash": "%s",
        					"height": 2
        				}
        			}
        		]
        	}
        }
	`, block1, block2, block3)
	assertQuery(t, query, response)
}

func TestBlocksTxsQuery(t *testing.T) {
	t.SkipNow()

	query := `
		{ 
			blocks(last: 1)   
			{  
				header
				{
					height
				}      
				transactions
				{
					txid 
					txtype 
				}
			}
	  	} 
		`
	response := fmt.Sprintf(`
	{
		"data": {
			"blocks": [
				{
					"header": {
						"height": 2
					},
					"transactions": [
						{
							"txid": "%s",
							"txtype": "3"
						}
					]
				}
			]
		}
	}
	`, bid3Hash)
	assertQuery(t, query, response)
}

func TestBlocksByDate(t *testing.T) {
	query := `
	{
	   blocks (since:  "1970-01-01T00:00:20+00:00" )     
		{
			 header
			 {
				height
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
        					"height": 1
        				}
        			}
        		]
        	}
        }
	`
	assertQuery(t, query, response)
}
