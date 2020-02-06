package query

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-wallet/v2/block"
	core "github.com/dusk-network/dusk-wallet/v2/transactions"
	"github.com/graphql-go/graphql"
)

var sc graphql.Schema
var db database.DB

func TestMain(m *testing.M) {

	// Setup lite DB
	_, db = lite.CreateDBConnection()
	defer db.Close()

	initializeDB(db)

	// Setup graphql Schema
	rootQuery := NewRoot(nil)
	sc, _ = graphql.NewSchema(
		graphql.SchemaConfig{Query: rootQuery.Query},
	)

	os.Exit(m.Run())
}

func initializeDB(db database.DB) {

	// Generate a dummy chain with a few blocks to test against
	chain := make([]*block.Block, 0)

	// Even random func is used, particular fields are hard-coded to make
	// comparision easier

	// block height 0
	t := &testing.T{}
	b1 := helper.RandomBlock(t, 0, 1)
	b1.Header.Hash, _ = hex.DecodeString("194dd13ee8a60ac017a82c41c0e2c02498d75f48754351072f392a085d469620")
	b1.Txs = make([]core.Transaction, 0)
	b1.Txs = append(b1.Txs, helper.FixedStandardTx(t, 0))
	b1.Header.Timestamp = 10
	chain = append(chain, b1)

	// block height 1
	b2 := helper.RandomBlock(t, 1, 1)
	b2.Header.Hash, _ = hex.DecodeString("9bf50e394bb81346f8b8db42bddd285ac344260c024a0df808baf7601417d748")
	b2.Txs = make([]core.Transaction, 0)
	b2.Txs = append(b2.Txs, helper.FixedStandardTx(t, 1))
	b2.Header.Timestamp = 20
	chain = append(chain, b2)

	// block height 2
	b3 := helper.RandomBlock(t, 2, 1)
	b3.Header.Hash, _ = hex.DecodeString("9467c5e774eb1b4825d08c0599a0b0815fca5dac16d9690026854ed8d1f229c9")
	b3.Txs = make([]core.Transaction, 0)
	b3.Txs = append(b3.Txs, helper.FixedStandardTx(t, 22))
	b3.Header.Timestamp = 30
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

func execute(query string, schema graphql.Schema, db database.DB) *graphql.Result {
	result := graphql.Do(graphql.Params{
		Schema:        schema,
		RequestString: query,
		Context:       context.WithValue(context.Background(), "database", db),
	})

	// Error check
	if len(result.Errors) > 0 {
		fmt.Printf("Unexpected errors inside ExecuteQuery: %v", result.Errors)
	}

	return result
}

func assertQuery(t *testing.T, query, response string) {
	result, err := json.MarshalIndent(execute(query, sc, db), "", "\t")
	if err != nil {
		t.Errorf("marshal response: %v", err)
	}

	equal, err := assertJSONs(result, []byte(response))
	if err != nil {
		t.Error(err)
	}

	t.Logf("Result:\n%s", result)
	if !equal {
		t.Error("expecting other response from this query")
	}
}

func assertJSONs(result, expected []byte) (bool, error) {

	var r interface{}
	if err := json.Unmarshal(result, &r); err != nil {
		return false, fmt.Errorf("mashalling error result val: %v", err)
	}

	var e interface{}
	if err := json.Unmarshal(expected, &e); err != nil {
		return false, fmt.Errorf("mashalling error expected val: %v", err)
	}

	return reflect.DeepEqual(r, e), nil
}
