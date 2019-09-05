package query

import (
	"encoding/hex"
	"time"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
)

var Block = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Block",
		Fields: graphql.Fields{
			"header": &graphql.Field{
				Type: Header,
			},
			"transactions": &graphql.Field{
				Type:    graphql.NewList(Transaction),
				Resolve: resolveTxs,
			},
		},
	},
)

var Header = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Header",
		Fields: graphql.Fields{
			"height": &graphql.Field{
				Type: graphql.Int,
			},
			"hash": &graphql.Field{
				Type: Hex,
			},
			"version": &graphql.Field{
				Type: graphql.String,
			},
			"prevblockhash": &graphql.Field{
				Type: Hex,
			},
			"seed": &graphql.Field{
				Type: Hex,
			},
			"txroot": &graphql.Field{
				Type: Hex,
			},
			"timestamp": &graphql.Field{
				Type: UnixTimestamp,
			},
		},
	},
)

var Transaction = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Transaction",
		Fields: graphql.Fields{
			"txid": &graphql.Field{
				Type: Hex,
			},
			"txtype": &graphql.Field{
				Type: graphql.String,
			},
			"blockhash": &graphql.Field{
				Type: Hex,
			},
		},
	},
)

var Hex = graphql.NewScalar(graphql.ScalarConfig{
	Name:        "Hex",
	Description: "Hex scalar type represents a byte array",
	// Serialize serializes `CustomID` to string.
	Serialize: func(value interface{}) interface{} {
		switch value := value.(type) {
		case []byte:
			return hex.EncodeToString(value)
		default:
			return nil
		}
	},
	// ParseValue parses GraphQL variables from `string` to `[]byte`.
	ParseValue: func(value interface{}) interface{} {
		switch value := value.(type) {
		case string:
			bytes, _ := hex.DecodeString(value)
			return bytes
		default:
			return nil
		}
	},
	// ParseLiteral parses GraphQL AST value to `CustomID`.
	ParseLiteral: func(valueAST ast.Value) interface{} {
		// not implemented
		return nil
	},
})

var UnixTimestamp = graphql.NewScalar(graphql.ScalarConfig{
	Name:        "UnixTimestamp",
	Description: "UnixTimestamp scalar type represents a unix time field",
	// Serialize serializes `CustomID` to string.
	Serialize: func(value interface{}) interface{} {
		switch value := value.(type) {
		case int64:
			tm := time.Unix(value, 0)
			return tm.String()
		default:
			return nil
		}
	},
	// ParseValue parses GraphQL variables from `string` to `[]byte`.
	ParseValue: func(value interface{}) interface{} {
		// not implemented
		return nil
	},
	// ParseLiteral parses GraphQL AST value to `CustomID`.
	ParseLiteral: func(valueAST ast.Value) interface{} {
		// not implemented
		return nil
	},
})
