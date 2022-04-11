// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package metrics

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
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

	//body, err := ioutil.ReadAll(resp.Body)
	//if err != nil {
	//	return err
	//}

	//bodyString := string(body)
	//
	//if err := json.Unmarshal(body, &target); err != nil {
	//	fmt.Println("ERROR executeQueryHTTP,  ", err, bodyString)
	//	return err
	//}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		body, _ := ioutil.ReadAll(resp.Body)

		fmt.Println("ERROR executeQueryHTTP,  ", err, string(body))
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
	// TODO: replace it with correct schema
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
	// TODO: replace it with correct schema
	var target interface{}

	return executeQuery(client, query, target, values)
}

//nolint
func getLatestBlock(duskInfo *DuskInfo, height uint64) (*Block, error) {
	query := fmt.Sprintf("{\"query\" : \"{  blocks(height: %d ) { header { hash height timestamp } transactions { txid txtype size } } }\"}", height)
	var resp map[string]interface{}

	err := executeQueryHTTP(duskInfo.GQLEndpoint, query, &resp)
	if err != nil {
		return nil, err
	}
	result := resp["data"]
	fmt.Println("getLatestBlock, ", result, resp)

	byt, err := json.Marshal(result)
	blocks := new(Blocks)
	if err := json.Unmarshal(byt, blocks); err != nil {
		return nil, err
	}

	if len(blocks.Blocks) == 0 {
		return nil, errors.New("block not found")
	}

	return &blocks.Blocks[0], nil
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
	// TODO: replace it with correct schema
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
			generatorblspublickey
		  }
		}
	  }
	`
	// TODO: replace it with correct schema
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
	// TODO: replace it with correct schema

	blk, err := executeQuery(duskInfo.GQLClient, query, new(Blocks), values)
	if err != nil {
		return nil, err
	}

	// log.Info("Got BlockByNumber", blk)

	return &blk.(*Blocks).Blocks[0], nil
}

func pendingTransactionCount(duskInfo *DuskInfo) (int, error) {
	query := "{\"query\" : \"{ mempool (txid: \\\"\\\") { txid txtype } }\"}"
	// TODO: replace it with correct schema
	var resp map[string]map[string][]map[string]string

	err := executeQueryHTTP(duskInfo.GQLEndpoint, query, &resp)
	if err != nil {
		return 0, err
	}

	count := 0

	result, ok := resp["data"]
	if ok {
		count = len(result["mempool"])
	}

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
	// TODO: replace it with correct schema
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
	// TODO: replace it with correct schema
	var target interface{}

	return executeQuery(client, query, target, values)
}
