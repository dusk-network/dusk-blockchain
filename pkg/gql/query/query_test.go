package query

import (
	"encoding/json"
	"fmt"
	"github.com/dusk-network/dusk-blockchain/pkg/core/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/crypto"
	"github.com/graphql-go/graphql"
	"os"
	"reflect"
	"testing"
)

var sc graphql.Schema

func TestMain(m *testing.M) {

	// Setup lite DB
	_, db := lite.CreateDBConnection()
	defer db.Close()

	initializeDB(db)

	// Setup graphql Schema
	rootQuery := NewRoot(nil, db)
	sc, _ = graphql.NewSchema(
		graphql.SchemaConfig{Query: rootQuery.Query},
	)

	// Setup mempool
	// TODO:

	os.Exit(m.Run())
}

func initializeDB(db database.DB) {

	// Generate a dummy chain with a few blocks to test against
	chain := make([]*block.Block, 0)

	// block height 0
	t := &testing.T{}
	b1 := helper.RandomBlock(t, 0, 1)
	b1.Header.Hash, _ = crypto.RandEntropy(32)
	chain = append(chain, b1)

	// block height 1
	b2 := helper.RandomBlock(t, 1, 1)
	b2.Header.Hash, _ = crypto.RandEntropy(32)
	chain = append(chain, b2)

	// block height 2
	b3 := helper.RandomBlock(t, 2, 1)
	b3.Header.Hash, _ = crypto.RandEntropy(32)
	chain = append(chain, b3)

	_ = db.Update(func(t database.Transaction) error {

		for _, block := range chain {
			err := t.StoreBlock(block)
			if err != nil {
				fmt.Print(err.Error())
				return err
			}
		}
		return nil
	})
}

func assertQuery(t *testing.T, query, response string) {
	result, err := json.MarshalIndent(Execute(query, sc), "", "\t")
	if err != nil {
		t.Errorf("marshal response: %v", err)
	}

	equal, err := assertJSONs(result, []byte(response))
	if err != nil {
		t.Error(err)
	}
	// t.Logf("Result:\n%s", result)
	if !equal {
		t.Error("expecting other response from this query")
	}
}

func assertJSONs(result, expected []byte) (bool, error) {
	var r interface{}
	var e interface{}

	var err error
	err = json.Unmarshal(result, &r)
	if err != nil {
		return false, fmt.Errorf("mashalling error result val: %v", err)
	}
	err = json.Unmarshal(expected, &e)
	if err != nil {
		return false, fmt.Errorf("mashalling error expected val: %v", err)
	}

	return reflect.DeepEqual(r, e), nil
}
