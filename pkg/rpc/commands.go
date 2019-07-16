package rpc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strconv"
	"time"

	cfg "gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database/heavy"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

// handler defines a method bound to an RPC command.
type handler func(*Server, []string) (string, error)

var (

	// rpcCmd maps method names to their actual functions.
	rpcCmd = map[string]handler{
		"version":       version,
		"ping":          pong,
		"uptime":        uptime,
		"getLastBlock":  getlastblock,
		"getMempoolTxs": getmempooltxs,
		// Publish Topic (experimental). Injects an event directly into EventBus system.
		// Would be useful on E2E testing. Mind the supportedTopics list when sends it
		"publishTopic": publishTopic,
		"exportData":   exportData,
	}

	// rpcAdminCmd holds all admin methods.
	rpcAdminCmd = map[string]bool{}

	// supported topics for injection into EventBus
	supportedTopics = [2]string{
		string(topics.Tx),
		string(topics.Block),
	}
)

// version will return the version of the client.
var version = func(s *Server, params []string) (string, error) {
	return protocol.NodeVer.String(), nil
}

// pong simply returns "pong" to let the caller know the server is up.
var pong = func(s *Server, params []string) (string, error) {
	return "pong", nil
}

// uptime returns the server uptime.
var uptime = func(s *Server, params []string) (string, error) {
	return strconv.FormatInt(time.Now().Unix()-s.startTime, 10), nil
}

var getlastblock = func(s *Server, params []string) (string, error) {

	r, err := s.rpcBus.Call(wire.GetLastBlock, wire.NewRequest(bytes.Buffer{}, 1))
	if err != nil {
		return "", err
	}

	b := &block.Block{}
	err = b.Decode(&r)
	if err != nil {
		return "", err
	}

	res, err := json.MarshalIndent(b, "", "\t")
	return string(res), err
}

var getmempooltxs = func(s *Server, params []string) (string, error) {

	r, err := s.rpcBus.Call(wire.GetMempoolTxs, wire.NewRequest(bytes.Buffer{}, 1))
	if err != nil {
		return "", err
	}

	lTxs, err := encoding.ReadVarInt(&r)
	if err != nil {
		return "", err
	}

	txs, err := transactions.FromReader(&r, lTxs)
	if err != nil {
		return "", err
	}

	res, err := json.MarshalIndent(txs, "", "\t")
	return string(res), err
}

var publishTopic = func(s *Server, params []string) (string, error) {

	if len(params) < 2 {
		return "", errors.New("expects always two input params - topics and payload bytes")
	}

	// Validate topic parameter.
	jsonrpcTopic := params[0]

	supported := false
	for _, topic := range supportedTopics {
		if topic == jsonrpcTopic {
			supported = true
			break
		}
	}

	if !supported {
		return "", fmt.Errorf("%s is not supported by publishTopic API", jsonrpcTopic)
	}

	payloadBuf := bytes.NewBufferString(params[1])
	s.eventBus.Publish(jsonrpcTopic, payloadBuf)

	res := fmt.Sprintf("published %s with len(payload) %d", jsonrpcTopic, len(params[1]))
	return res, nil
}

var exportData = func(s *Server, params []string) (string, error) {

	_, db := heavy.CreateDBConnection()

	optionTransactions := false
	if len(params) > 0 {
		if params[0] == "includeBlockTxs" {
			optionTransactions = true
		}
	}

	var chain []block.Block
	err := db.View(func(t database.Transaction) error {
		var height uint64
		for {

			hash, err := t.FetchBlockHashByHeight(height)
			if err != nil {
				return nil
			}

			header, err := t.FetchBlockHeader(hash)
			if err != nil {
				return err
			}

			txs := make([]transactions.Transaction, 0)
			if optionTransactions {
				txs, err = t.FetchBlockTxs(hash)
				if err != nil {
					return err
				}
			}
			b := block.Block{
				Header: header,
				Txs:    txs,
			}

			chain = append(chain, b)
			height++
		}
	})

	// encode as json
	encoded, _ := json.MarshalIndent(chain, "", "\t")

	// Export to a temp file
	filePath := fmt.Sprintf("/tmp/dusk-node_%s.json", cfg.Get().RPC.Port)
	_ = ioutil.WriteFile(filePath, encoded, 0644)

	return fmt.Sprintf("\"exported to %s\"", filePath), err
}
