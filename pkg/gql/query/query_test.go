package query

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	core "github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/graphql-go/graphql"
	assert "github.com/stretchr/testify/require"
)

var sc graphql.Schema
var db database.DB

func TestMain(m *testing.M) {

	// Setup lite DB
	_, db = lite.CreateDBConnection()
	defer func() {
		_ = db.Close()
	}()

	if err := initializeDB(db); err != nil {
		panic(err)
	}

	// Setup graphql Schema
	rootQuery := NewRoot(nil)
	sc, _ = graphql.NewSchema(
		graphql.SchemaConfig{Query: rootQuery.Query},
	)

	os.Exit(m.Run())
}

var block1 = "194dd13ee8a60ac017a82c41c0e2c02498d75f48754351072f392a085d469620"
var block2 = "9bf50e394bb81346f8b8db42bddd285ac344260c024a0df808baf7601417d748"
var block3 = "9467c5e774eb1b4825d08c0599a0b0815fca5dac16d9690026854ed8d1f229c9"

var bid1 = core.MockDeterministicBid(100, 100000, make([]byte, 32), make([]byte, 33))
var bid2 = core.MockDeterministicBid(350, 1239013, make([]byte, 32), make([]byte, 32))
var bid3 = core.MockDeterministicBid(200, 100002, make([]byte, 32), make([]byte, 33))

var bid1HashB, _ = bid1.CalculateHash()
var bid2HashB, _ = bid2.CalculateHash()
var bid3HashB, _ = bid3.CalculateHash()

var bid1Hash = hex.EncodeToString(bid1HashB)
var bid2Hash = hex.EncodeToString(bid2HashB)
var bid3Hash = hex.EncodeToString(bid3HashB)

func initializeDB(db database.DB) error {

	// Generate a dummy chain with a few blocks to test against
	chain := make([]*block.Block, 0)

	// Even if random func is used, particular fields are hard-coded to make
	// comparison easier

	// block height 0
	b1 := helper.RandomBlock(0, 1)
	b1.Header.Hash, _ = hex.DecodeString(block1)
	b1.Txs = make([]core.ContractCall, 0)
	b1.Txs = append(b1.Txs, bid1)
	_, err := b1.Txs[0].CalculateHash()
	if err != nil {
		return err
	}

	b1.Header.Timestamp = 10
	chain = append(chain, b1)

	// block height 1
	b2 := helper.RandomBlock(1, 1)
	b2.Header.Hash, _ = hex.DecodeString(block2)
	b2.Txs = make([]core.ContractCall, 0)
	b2.Txs = append(b2.Txs, bid2)
	b2.Header.Timestamp = 20
	chain = append(chain, b2)

	// block height 2
	b3 := helper.RandomBlock(2, 1)
	b3.Header.Hash, _ = hex.DecodeString(block3)
	b3.Txs = make([]core.ContractCall, 0)
	b3.Txs = append(b3.Txs, bid3)
	b3.Header.Timestamp = 30
	chain = append(chain, b3)

	return db.Update(func(t database.Transaction) error {

		for _, block := range chain {
			err := t.StoreBlock(block)
			if err != nil {
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
		Context:       context.WithValue(context.Background(), "database", db), //nolint
	})

	// Error check
	if len(result.Errors) > 0 {
		fmt.Printf("Unexpected errors inside ExecuteQuery: %v", result.Errors)
	}

	return result
}

func assertQuery(t *testing.T, query, response string) {
	assert := assert.New(t)
	result, err := json.MarshalIndent(execute(query, sc, db), "", "\t")
	assert.NoError(err)

	equal, err := assertJSONs(result, []byte(response))
	assert.NoError(err)
	assert.True(equal)
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
