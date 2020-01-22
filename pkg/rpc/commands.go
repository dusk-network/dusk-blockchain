package rpc

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-wallet/wallet"
)

// handler defines a method bound to an RPC command.
type handler func(*Server, []string) (string, error)

var (

	// Commands maps method names to their actual functions.
	Commands = map[string]handler{
		"transfer":             transfer,
		"bid":                  sendBidTx,
		"stake":                sendStakeTx,
		"createwallet":         createWallet,
		"loadwallet":           loadWallet,
		"createfromseed":       createFromSeed,
		"address":              address,
		"balance":              balance,
		"unconfirmedbalance":   unconfirmedBalance,
		"txhistory":            txHistory,
		"syncprogress":         syncProgress,
		"automateconsensustxs": automateConsensusTxs,
		"walletstatus":         walletStatus,
		"rebuildchain":         rebuildChain,
		"viewmempool":          viewMempool,

		// Publish Topic (experimental). Injects an event directly into EventBus system.
		// Would be useful on E2E testing. Mind the supportedTopics list when sends it
		"publishTopic": publishTopic,
	}

	// rpcAdminCmd holds all admin methods.
	rpcAdminCmd = map[string]bool{
		"transfer":       true,
		"bid":            true,
		"stake":          true,
		"createwallet":   true,
		"loadwallet":     true,
		"createFromSeed": true,
		"publishTopic":   true,
	}

	// supported topics for injection into EventBus
	supportedTopics = [3]topics.Topic{
		topics.Tx,
		topics.Block,
		topics.StopConsensus,
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
		if topic.String() == jsonrpcTopic {
			supported = true
			break
		}
	}

	if !supported {
		return "", fmt.Errorf("%s is not supported by publishTopic API", jsonrpcTopic)
	}

	payload, _ := hex.DecodeString(params[1])
	rpcTopic := topics.StringToTopic(jsonrpcTopic)
	s.eventBus.Publish(rpcTopic, bytes.NewBuffer(payload))

	result :=
		`{ 
			"result": "published"
		}`
	return result, nil
}

var sendBidTx = func(s *Server, params []string) (string, error) {
	if len(params) < 2 {
		return "", fmt.Errorf("missing parameters: amount/locktime")
	}

	amount, err := strconv.Atoi(params[0])
	if err != nil {
		return "", fmt.Errorf("converting amount string to an integer: %v", err)
	}

	lockTime, err := strconv.Atoi(params[1])
	if err != nil {
		return "", fmt.Errorf("converting locktime string to an integer: %v", err)
	}

	buf := new(bytes.Buffer)
	if err := rpcbus.MarshalConsensusTxRequest(buf, uint64(amount), uint64(lockTime)); err != nil {
		return "", err
	}

	txid, err := s.rpcBus.Call(rpcbus.SendBidTx, rpcbus.NewRequest(*buf), 0)
	if err != nil {
		return "", err
	}

	result := fmt.Sprintf("Success! TxID: %s", hex.EncodeToString(txid.Bytes()))
	return result, err
}

var transfer = func(s *Server, params []string) (string, error) {
	if len(params) < 2 {
		return "", errors.New("not enough parameters")
	}

	amount, err := strconv.Atoi(params[0])
	if err != nil {
		return "", fmt.Errorf("converting amount string to an integer: %v", err)
	}

	address := params[1]

	buf := new(bytes.Buffer)
	if err := encoding.WriteUint64LE(buf, uint64(amount)); err != nil {
		return "", fmt.Errorf("error writing amount to buffer: %v", err)
	}

	if err := encoding.WriteString(buf, address); err != nil {
		return "", fmt.Errorf("error writing address to buffer: %v", err)
	}

	txid, err := s.rpcBus.Call(rpcbus.SendStandardTx, rpcbus.NewRequest(*buf), 0)
	if err != nil {
		return "", err
	}

	idString, err := encoding.ReadString(&txid)
	if err != nil {
		return "", err
	}

	result := fmt.Sprintf("Success! TxID: %s", hex.EncodeToString([]byte(idString)))
	return result, err
}

var sendStakeTx = func(s *Server, params []string) (string, error) {
	if len(params) < 2 {
		return "", errors.New("not enough parameters")
	}

	amount, err := strconv.Atoi(params[0])
	if err != nil {
		return "", fmt.Errorf("error converting amount string to an integer: %v", err)
	}

	lockTime, err := strconv.Atoi(params[1])
	if err != nil {
		return "", fmt.Errorf("error converting locktime string to an integer: %v", err)
	}

	buf := new(bytes.Buffer)
	if err := encoding.WriteUint64LE(buf, uint64(amount)); err != nil {
		return "", fmt.Errorf("error writing amount to buffer: %v", err)
	}

	if err := encoding.WriteUint64LE(buf, uint64(lockTime)); err != nil {
		return "", fmt.Errorf("error writing address to buffer: %v", err)
	}

	txid, err := s.rpcBus.Call(rpcbus.SendStakeTx, rpcbus.NewRequest(*buf), 0)
	if err != nil {
		return "", err
	}

	idString, err := encoding.ReadString(&txid)
	if err != nil {
		return "", err
	}

	result := fmt.Sprintf("Success! TxID: %s", hex.EncodeToString([]byte(idString)))
	return result, err
}

var createWallet = func(s *Server, params []string) (string, error) {
	if len(params) < 1 {
		return "", fmt.Errorf("missing parameter: password")
	}

	buf := new(bytes.Buffer)
	if err := encoding.WriteString(buf, params[0]); err != nil {
		return "", err
	}

	pubKeyBuf, err := s.rpcBus.Call(rpcbus.CreateWallet, rpcbus.NewRequest(*buf), 0)
	if err != nil {
		return "", err
	}

	pubKey, err := encoding.ReadString(&pubKeyBuf)
	if err != nil {
		return "", err
	}

	result := fmt.Sprintf("Wallet created.\nYour address is %s", pubKey)
	return result, err
}

var loadWallet = func(s *Server, params []string) (string, error) {
	if len(params) < 1 {
		return "", fmt.Errorf("missing parameter: password")
	}

	buf := new(bytes.Buffer)
	if err := encoding.WriteString(buf, params[0]); err != nil {
		return "", err
	}

	pubKeyBuf, err := s.rpcBus.Call(rpcbus.LoadWallet, rpcbus.NewRequest(*buf), 0)
	if err != nil {
		return "", err
	}

	pubKey, err := encoding.ReadString(&pubKeyBuf)
	if err != nil {
		return "", err
	}

	result := fmt.Sprintf("Wallet loaded.\nYour address is %s", pubKey)
	return result, err
}

var createFromSeed = func(s *Server, params []string) (string, error) {
	if len(params) < 2 {
		return "", fmt.Errorf("not enough parameters provided")
	}

	buf := new(bytes.Buffer)
	// seed
	if err := encoding.Write512(buf, []byte(params[0])); err != nil {
		return "", err
	}

	// password
	if err := encoding.WriteString(buf, params[1]); err != nil {
		return "", err
	}

	pubKeyBuf, err := s.rpcBus.Call(rpcbus.CreateFromSeed, rpcbus.NewRequest(*buf), 0)
	if err != nil {
		return "", err
	}

	pubKey, err := encoding.ReadString(&pubKeyBuf)
	if err != nil {
		return "", err
	}

	result := fmt.Sprintf("Wallet created! Your address is %s", pubKey)
	return result, err
}

var address = func(s *Server, params []string) (string, error) {
	addressBuf, err := s.rpcBus.Call(rpcbus.GetAddress, rpcbus.NewRequest(bytes.Buffer{}), 0)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Your address is %s", addressBuf.String()), nil
}

var balance = func(s *Server, params []string) (string, error) {
	balanceBuf, err := s.rpcBus.Call(rpcbus.GetBalance, rpcbus.NewRequest(bytes.Buffer{}), 0)
	if err != nil {
		return "", err
	}

	var unlockedBalance, lockedBalance uint64
	if err := encoding.ReadUint64LE(&balanceBuf, &unlockedBalance); err != nil {
		return "", err
	}

	if err := encoding.ReadUint64LE(&balanceBuf, &lockedBalance); err != nil {
		return "", err
	}

	result := fmt.Sprintf("Unlocked balance: %.8f\nLocked balance: %.8f", float64(unlockedBalance)/float64(wallet.DUSK), float64(lockedBalance)/float64(wallet.DUSK))
	return result, nil
}

var unconfirmedBalance = func(s *Server, params []string) (string, error) {
	balanceBuf, err := s.rpcBus.Call(rpcbus.GetUnconfirmedBalance, rpcbus.NewRequest(bytes.Buffer{}), 0)
	if err != nil {
		return "", err
	}

	var unconfirmedBalance uint64
	if err := encoding.ReadUint64LE(&balanceBuf, &unconfirmedBalance); err != nil {
		return "", err
	}

	result := fmt.Sprintf("Unconfirmed balance: %.8f", float64(unconfirmedBalance)/float64(wallet.DUSK))
	return result, nil
}

var txHistory = func(s *Server, params []string) (string, error) {
	txRecordsBuf, err := s.rpcBus.Call(rpcbus.GetTxHistory, rpcbus.Request{bytes.Buffer{}, make(chan rpcbus.Response, 1)}, 5*time.Second)
	if err != nil {
		return "", err
	}

	return txRecordsBuf.String(), nil
}

var automateConsensusTxs = func(s *Server, params []string) (string, error) {
	if _, err := s.rpcBus.Call(rpcbus.AutomateConsensusTxs, rpcbus.Request{bytes.Buffer{}, make(chan rpcbus.Response, 1)}, 5*time.Second); err != nil {
		return "", err
	}

	return "Consensus transactions are now being automated -- you can update your settings in the dusk.toml config file", nil
}

var syncProgress = func(s *Server, params []string) (string, error) {
	percentageBuf, err := s.rpcBus.Call(rpcbus.GetSyncProgress, rpcbus.Request{bytes.Buffer{}, make(chan rpcbus.Response, 1)}, 2*time.Second)
	if err != nil {
		return "", err
	}

	return percentageBuf.String(), nil
}

var walletStatus = func(s *Server, params []string) (string, error) {
	walletStatusBuf, err := s.rpcBus.Call(rpcbus.IsWalletLoaded, rpcbus.Request{bytes.Buffer{}, make(chan rpcbus.Response, 1)}, 2*time.Second)
	if err != nil {
		return "", err
	}

	var status bool
	if err := encoding.ReadBool(&walletStatusBuf, &status); err != nil {
		return "", err
	}

	return fmt.Sprintf("%v", status), nil
}

var rebuildChain = func(s *Server, params []string) (string, error) {
	if _, err := s.rpcBus.Call(rpcbus.RebuildChain, rpcbus.Request{bytes.Buffer{}, make(chan rpcbus.Response, 1)}, 0*time.Second); err != nil {
		return "", err
	}

	return "Chain reset complete. Starting sync...", nil
}

var viewMempool = func(s *Server, params []string) (string, error) {
	// Encode filtering information
	var buf bytes.Buffer
	if len(params) > 1 {
		buf = *bytes.NewBuffer([]byte(params[0]))
	}

	txsBuf, err := s.rpcBus.Call(rpcbus.GetMempoolView, rpcbus.Request{buf, make(chan rpcbus.Response, 1)}, 2*time.Second)
	if err != nil {
		return "", err
	}

	return txsBuf.String(), nil
}
