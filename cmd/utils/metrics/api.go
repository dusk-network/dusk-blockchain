package metrics

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/machinebox/graphql"
)

func executeQuery(client *graphql.Client, query string, target interface{}, values map[string]interface{}) (interface{}, error) {
	req := graphql.NewRequest(query)

	if len(values) > 0 {
		for k, v := range values {
			req.Var(k, v)
		}
	}

	// define a Context for the request
	ctx := context.Background()

	// run it and capture the response
	if err := client.Run(ctx, req, &target); err != nil {
		return nil, err
	}

	return target, nil
}

func executeQueryHTTP(endpoint string, query string, target interface{}) error {

	buf := bytes.Buffer{}
	if _, err := buf.Write([]byte(query)); err != nil {
		return errors.New("invalid query")
	}

	//nolint:gosec
	resp, err := http.Post(endpoint, "application/json", &buf)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return err
	}

	return nil
}

//nolint
func getLatestTransactions(client *graphql.Client, values map[string]interface{}) (interface{}, error) {
	query := `
	  query {
		transactions(last: 15) {
			txid
			blockhash
		}
	  }
	`
	//TODO: replace it with correct schema
	var target interface{}

	return executeQuery(client, query, target, values)
}

//nolint
func getLatestBlocks(client *graphql.Client, values map[string]interface{}) (interface{}, error) {
	query := `
	  query {
		blocks(last: 15) {
		  header {
			hash
			height
			timestamp
		  }
		}
	  }
	`
	//TODO: replace it with correct schema
	var target interface{}

	return executeQuery(client, query, target, values)
}

//nolint
func getBlockTransactionsByHash(client *graphql.Client, values map[string]interface{}) (interface{}, error) {
	query := `
	  query ($hash: String!) {
		blocks(hash: $hash) {
		  transactions {
			txid
			txtype
			size
		  }
		}
	  }
	`
	//TODO: replace it with correct schema
	var target interface{}

	return executeQuery(client, query, target, values)
}

//nolint
func getBlockByHash(client *graphql.Client, values map[string]interface{}) (interface{}, error) {
	query := `
	  query($hash: String!) {
		blocks(hash: $hash ) {
		  header {
			hash
			height
			timestamp
			version
			seed
			prevblockhash
			txroot
		  }
		}
	  }
	`
	//TODO: replace it with correct schema
	var target interface{}

	return executeQuery(client, query, target, values)
}

func getBlockByNumber(duskInfo *DuskInfo, values map[string]interface{}) (*Block, error) {
	query := `
	  query($height: Int!) {
		blocks(height: $height) {
		  header {
			hash
			height
			timestamp
		  }
		  transactions {
			txid
			txtype
			size
		  }
		}
	  }
	`
	//TODO: replace it with correct schema

	blk, err := executeQuery(duskInfo.GQLClient, query, new(Blocks), values)
	if err != nil {
		return nil, err
	}

	//log.Info("Got BlockByNumber", blk)

	return &blk.(*Blocks).Blocks[0], nil
}

func pendingTransactionCount(duskInfo *DuskInfo) (int, error) {
	query := "{\"query\" : \"{ mempool (txid: \\\"\\\") { txid txtype } }\"}"
	//TODO: replace it with correct schema
	var resp map[string]map[string][]map[string]string

	err := executeQueryHTTP(duskInfo.GQLEndpoint, query, &resp)
	if err != nil {
		return 0, err
	}

	result, ok := resp["data"]
	count := 0
	if ok {
		count = len(result["mempool"])
	}

	//fmt.Println("Got PendingTransactionCount", resp, count)
	return count, nil
}

//nolint
func getTransactionByID(client *graphql.Client, values map[string]interface{}) (interface{}, error) {
	query := `
	  query($txid: String!) {
		transactions(txid: $txid) {
		  txid
		  blockhash
		  txtype
		  size
		  output {
			pubkey
		  }
		  input {
			keyimage
		  }
		}
	  }
	`
	//TODO: replace it with correct schema
	var target interface{}

	return executeQuery(client, query, target, values)
}

//nolint
func getBlocksCountQuery(client *graphql.Client, values map[string]interface{}) (interface{}, error) {
	query := `
	  query($time: DateTime!) {
		tip: blocks(height: -1) {
		  header {
			height
		  }
		}
		old: blocks(since: $time) {
		  header {
			height
		  }
		}
	  }
	`
	//TODO: replace it with correct schema
	var target interface{}

	return executeQuery(client, query, target, values)
}
