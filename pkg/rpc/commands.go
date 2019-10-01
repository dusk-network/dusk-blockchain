package rpc

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
)

// handler defines a method bound to an RPC command.
type handler func(*Server, []string) (string, error)

var (

	// rpcCmd maps method names to their actual functions.
	rpcCmd = map[string]handler{
		// Publish Topic (experimental). Injects an event directly into EventBus system.
		// Would be useful on E2E testing. Mind the supportedTopics list when sends it
		"publishTopic": publishTopic,
		"sendBidTx":    sendBidTx,
	}

	// rpcAdminCmd holds all admin methods.
	rpcAdminCmd = map[string]bool{}

	// supported topics for injection into EventBus
	supportedTopics = [2]string{
		string(topics.Tx),
		string(topics.Block),
	}
)

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

	payload, _ := hex.DecodeString(params[1])
	s.eventBus.Publish(jsonrpcTopic, bytes.NewBuffer(payload))

	result := "{ \"result\": \"published\" }"
	return result, nil
}

var sendBidTx = func(s *Server, params []string) (string, error) {

	// TODO: Not Implemented

	result := "{ \"txid\": \"unknown\" }"
	return result, nil
}
