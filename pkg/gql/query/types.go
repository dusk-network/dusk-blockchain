// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package query

import (
	"encoding/hex"
	"time"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
)

// Block is the graphql object representing blocks.
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

// Header is the graphql object representing block header.
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
			"statehash": &graphql.Field{
				Type: Hex,
			},
			"seed": &graphql.Field{
				Type: Hex,
			},
			"generatorblspubkey": &graphql.Field{
				Type: Hex,
			},
			"timestamp": &graphql.Field{
				Type: UnixTimestamp,
			},
			"reward": &graphql.Field{
				Type:    graphql.Float,
				Resolve: resolveReward,
			},
			"feespaid": &graphql.Field{
				Type:    graphql.Float,
				Resolve: resolveFee,
			},
			"step": &graphql.Field{
				Type:    graphql.Int,
				Resolve: resolveStep,
			},
		},
	},
)

// Transaction is the graphql object representing transactions.
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
			"blocktimestamp": &graphql.Field{
				Type: UnixTimestamp,
			},
			"output": &graphql.Field{
				Type: graphql.NewList(Output),
			},
			"input": &graphql.Field{
				Type: graphql.NewList(Input),
			},
			"score": &graphql.Field{
				Type: Hex,
			},
			"size": &graphql.Field{
				Type: graphql.Int,
			},
			"gaslimit": &graphql.Field{
				Type: graphql.Float,
			},
			"gasprice": &graphql.Field{
				Type: graphql.Int,
			},
			"gasspent": &graphql.Field{
				Type: graphql.Float,
			},
			"json": &graphql.Field{
				Type: graphql.String,
			},
			"txerror": &graphql.Field{
				Type: graphql.String,
			},
		},
	},
)

// Output is the graphql object representing output.
var Output = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Output",
		Fields: graphql.Fields{
			"pubkey": &graphql.Field{
				Type: Hex,
			},
		},
	},
)

// Input is the graphql object representing input.
var Input = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Input",
		Fields: graphql.Fields{
			"keyimage": &graphql.Field{
				Type: Hex,
			},
		},
	},
)

// Hex is the graphql object representing a hex scalar.
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

// UnixTimestamp the one and only.
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
