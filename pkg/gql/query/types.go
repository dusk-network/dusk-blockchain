package query

import (
	"encoding/base64"
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
			// TODO: write our own Scalar to handle uint64
			"height": &graphql.Field{
				Type: graphql.Int,
			},
			"hash": &graphql.Field{
				Type: Base64,
			},
			"version": &graphql.Field{
				Type: graphql.Int,
			},
			"prevblockhash": &graphql.Field{
				Type: Base64,
			},
			"seed": &graphql.Field{
				Type: Base64,
			},
			"txroot": &graphql.Field{
				Type: Base64,
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
				Type: Base64,
			},
			"txtype": &graphql.Field{
				Type: graphql.String,
			},
			"blockhash": &graphql.Field{
				Type: Base64,
			},
		},
	},
)

var Base64 = graphql.NewScalar(graphql.ScalarConfig{
	Name:        "Base64",
	Description: "Base64 scalar type represents a byte array",
	// Serialize serializes `CustomID` to string.
	Serialize: func(value interface{}) interface{} {
		switch value := value.(type) {
		case []byte:
			return base64.StdEncoding.EncodeToString(value)
		default:
			return nil
		}
	},
	// ParseValue parses GraphQL variables from `string` to `[]byte`.
	ParseValue: func(value interface{}) interface{} {
		switch value := value.(type) {
		case string:
			bytes, _ := base64.StdEncoding.DecodeString(value)
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
