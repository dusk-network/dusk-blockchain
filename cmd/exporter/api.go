package main

import (
	"context"
	"github.com/machinebox/graphql"
)

func ExecuteQuery(client *graphql.Client, query string, target interface{}, values map[string]interface{}) (interface{}, error) {
	req := graphql.NewRequest(query)

	if values != nil && len(values) > 0 {
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

func GetLatestTransactions(client *graphql.Client, values map[string]interface{}) (interface{}, error) {
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

	return ExecuteQuery(client, query, target, values)
}

func GetLatestBlocks(client *graphql.Client, values map[string]interface{}) (interface{}, error) {
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

	return ExecuteQuery(client, query, target, values)
}

func GetBlockTransactionsByHash(client *graphql.Client, values map[string]interface{}) (interface{}, error) {
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

	return ExecuteQuery(client, query, target, values)
}

func GetBlockByHash(client *graphql.Client, values map[string]interface{}) (interface{}, error) {
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

	return ExecuteQuery(client, query, target, values)
}

func GetBlockByNumber(client *graphql.Client, values map[string]interface{}) (*Block, error) {
	query := `
	  query($height: Int!) {
		blocks(height: $height) {
		  header {
			hash
			height
			timestamp
		  }
		}
	  }
	`
	//TODO: replace it with correct schema

	blk, err := ExecuteQuery(client, query, new(Blocks), values)
	if err != nil {
		return nil, err
	}

	//log.Info("Got BlockByNumber", blk)

	return &blk.(*Blocks).Blocks[0], nil
}

func PendingTransactionCount(client *graphql.Client, values map[string]interface{}) (int, error) {
	query := `
	mempool(txid: "") {
		txid
		txtype
 	 },
	`
	//TODO: replace it with correct schema
	var resp map[string]map[string][]map[string]string
	txs, err := ExecuteQuery(client, query, resp, values)
	if err != nil {
		return 0, err
	}

	//log.Info("Got PendingTransactionCount", txs)

	result, ok := txs.(map[string]map[string][]map[string]string)["data"]
	count := 0
	if ok {
		count = len(result["transactions"])
	}
	return count, nil
}

func GetTransactionByID(client *graphql.Client, values map[string]interface{}) (interface{}, error) {
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

	return ExecuteQuery(client, query, target, values)
}

func GetBlocksCountQuery(client *graphql.Client, values map[string]interface{}) (interface{}, error) {
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

	return ExecuteQuery(client, query, target, values)
}
