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
