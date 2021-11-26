// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package topics

import (
	"bytes"
	"fmt"
	"io"
)

// Topic defines a topic.
type Topic uint8

// A list of all valid topics.
const (
	// Standard topics.
	Version Topic = iota
	VerAck
	Ping
	Pong

	// Seeder communications topics.
	Challenge
	Response
	GetAddrs
	Addr

	// Data exchange topics.
	GetData
	GetBlocks
	Tx
	Block
	AcceptedBlock
	MemPool
	Inv

	// Gossiped topics.
	Candidate
	NewBlock
	Reduction
	Agreement

	// Peer topics.
	Gossip

	// Error topics.
	Unknown
	Reject

	// Internal.
	Quit
	Log
	Monitor
	Test

	// RPCBus topics.
	GetMempoolTxs
	GetMempoolTxsBySize
	SendMempoolTx
	VerifyStateTransition
	ExecuteStateTransition

	// Cross-process RPCBus topics.
	GetMempoolView
	SendBidTx
	SendStakeTx
	SendStandardTx
	GetBalance
	GetUnconfirmedBalance
	GetAddress
	GetTxHistory
	AutomateConsensusTxs
	GetSyncProgress
	RebuildChain
	ClearWalletDatabase
	StartProfile
	StopProfile

	// Cross-network RPCBus topics.
	GetCandidate

	// Monitoring topics.
	SyncProgress

	// Kadcast wire messaging.
	Kadcast

	// Kadcast wire point-to-point messaging.
	KadcastPoint
)

// Priority topics should be handled in an even proportion to ensure network
// critical messages are processed in a timely fashion
var Priority = []Topic{
	// consensus messages need to be responded to quickly to improve network convergence
	Candidate,
	NewBlock,
	Reduction,
	Agreement,
	// these will also be priority because they are anyway hard to spam
	Kadcast,
	KadcastPoint,
	// if a new block has arrived, this is a priority event as it needs
	// to be attached to the chain as soon as possible in order to mine
	AcceptedBlock,
}

type topicBuf struct {
	Topic
	bytes.Buffer
	str string
}

// Topics represents the associated string and byte representation respectively
// of the Topic objects.
// NOTE: this needs to be in the same order in which the topics are declared.
var Topics = [...]topicBuf{
	{Version, *(bytes.NewBuffer([]byte{byte(Version)})), "version"},
	{VerAck, *(bytes.NewBuffer([]byte{byte(VerAck)})), "verack"},
	{Ping, *(bytes.NewBuffer([]byte{byte(Ping)})), "ping"},
	{Pong, *(bytes.NewBuffer([]byte{byte(Pong)})), "pong"},
	{Challenge, *(bytes.NewBuffer([]byte{byte(Challenge)})), "challenge"},
	{Response, *(bytes.NewBuffer([]byte{byte(Response)})), "response"},
	{GetAddrs, *(bytes.NewBuffer([]byte{byte(GetAddrs)})), "getaddrs"},
	{Addr, *(bytes.NewBuffer([]byte{byte(Addr)})), "addr"},
	{GetData, *(bytes.NewBuffer([]byte{byte(GetData)})), "getdata"},
	{GetBlocks, *(bytes.NewBuffer([]byte{byte(GetBlocks)})), "getblocks"},
	{Tx, *(bytes.NewBuffer([]byte{byte(Tx)})), "tx"},
	{Block, *(bytes.NewBuffer([]byte{byte(Block)})), "block"},
	{AcceptedBlock, *(bytes.NewBuffer([]byte{byte(AcceptedBlock)})), "acceptedblock"},
	{MemPool, *(bytes.NewBuffer([]byte{byte(MemPool)})), "mempool"},
	{Inv, *(bytes.NewBuffer([]byte{byte(Inv)})), "inv"},
	{Candidate, *(bytes.NewBuffer([]byte{byte(Candidate)})), "candidate"},
	{NewBlock, *(bytes.NewBuffer([]byte{byte(NewBlock)})), "newblock"},
	{Reduction, *(bytes.NewBuffer([]byte{byte(Reduction)})), "reduction"},
	{Agreement, *(bytes.NewBuffer([]byte{byte(Agreement)})), "agreement"},
	{Gossip, *(bytes.NewBuffer([]byte{byte(Gossip)})), "gossip"},
	{Unknown, *(bytes.NewBuffer([]byte{byte(Unknown)})), "unknown"},
	{Reject, *(bytes.NewBuffer([]byte{byte(Reject)})), "reject"},
	{Quit, *(bytes.NewBuffer([]byte{byte(Quit)})), "quit"},
	{Log, *(bytes.NewBuffer([]byte{byte(Log)})), "log"},
	{Monitor, *(bytes.NewBuffer([]byte{byte(Log)})), "monitor_topic"},
	{Test, *(bytes.NewBuffer([]byte{byte(Test)})), "__test"},
	{GetMempoolTxs, *(bytes.NewBuffer([]byte{byte(GetMempoolTxs)})), "getmempooltxs"},
	{GetMempoolTxsBySize, *(bytes.NewBuffer([]byte{byte(GetMempoolTxsBySize)})), "getmempooltxsbysize"},
	{SendMempoolTx, *(bytes.NewBuffer([]byte{byte(SendMempoolTx)})), "sendmempooltx"},
	{VerifyStateTransition, *(bytes.NewBuffer([]byte{byte(VerifyStateTransition)})), "validatestatetransition"},
	{ExecuteStateTransition, *(bytes.NewBuffer([]byte{byte(ExecuteStateTransition)})), "executestatetransition"},
	{GetMempoolView, *(bytes.NewBuffer([]byte{byte(GetMempoolView)})), "getmempoolview"},
	{SendBidTx, *(bytes.NewBuffer([]byte{byte(SendBidTx)})), "sendbidtx"},
	{SendStakeTx, *(bytes.NewBuffer([]byte{byte(SendStakeTx)})), "sendstaketx"},
	{SendStandardTx, *(bytes.NewBuffer([]byte{byte(SendStandardTx)})), "sendstandardtx"},
	{GetBalance, *(bytes.NewBuffer([]byte{byte(GetBalance)})), "getbalance"},
	{GetUnconfirmedBalance, *(bytes.NewBuffer([]byte{byte(GetUnconfirmedBalance)})), "getunconfirmedbalance"},
	{GetAddress, *(bytes.NewBuffer([]byte{byte(GetAddress)})), "getaddress"},
	{GetTxHistory, *(bytes.NewBuffer([]byte{byte(GetTxHistory)})), "gettxhistory"},
	{AutomateConsensusTxs, *(bytes.NewBuffer([]byte{byte(AutomateConsensusTxs)})), "automateconsensustxs"},
	{GetSyncProgress, *(bytes.NewBuffer([]byte{byte(GetSyncProgress)})), "getsyncprogress"},
	{RebuildChain, *(bytes.NewBuffer([]byte{byte(RebuildChain)})), "rebuildchain"},
	{ClearWalletDatabase, *(bytes.NewBuffer([]byte{byte(ClearWalletDatabase)})), "clearwalletdatabase"},
	{StartProfile, *(bytes.NewBuffer([]byte{byte(StartProfile)})), "startprofile"},
	{StopProfile, *(bytes.NewBuffer([]byte{byte(StopProfile)})), "stopprofile"},
	{GetCandidate, *(bytes.NewBuffer([]byte{byte(GetCandidate)})), "getcandidate"},
	{SyncProgress, *(bytes.NewBuffer([]byte{byte(SyncProgress)})), "syncprogress"},
	{Kadcast, *(bytes.NewBuffer([]byte{byte(Kadcast)})), "kadcast"},
}

func checkConsistency(topics []topicBuf) {
	for i, topic := range topics {
		if uint8(topic.Topic) != uint8(i) {
			panic(fmt.Errorf("mismatch detected between a topic and its index. Please check the `topicBuf` array at index: %d", i))
		}
	}
}

func init() {
	checkConsistency(Topics[:])
}

// ToBuffer returns Topic as a Buffer.
func (t Topic) ToBuffer() bytes.Buffer {
	return Topics[int(t)].Buffer
}

// String representation of a known topic.
func (t Topic) String() string {
	if len(Topics) > int(t) {
		return Topics[t].str
	}

	return "unknown"
}

// StringToTopic turns a string into a Topic if the Topic is in the enum of known topics.
// Return Unknown topic if the string is not coupled with any of them.
func StringToTopic(topic string) Topic {
	for _, t := range Topics {
		if t.Topic.String() == topic {
			return t.Topic
		}
	}

	return Unknown
}

// Prepend a topic to a binary-serialized form of a message.
func Prepend(b *bytes.Buffer, t Topic) error {
	var buf bytes.Buffer
	if int(t) > len(Topics) {
		buf = *(bytes.NewBuffer([]byte{byte(t)}))
	} else {
		buf = Topics[int(t)].Buffer
	}

	if _, err := b.WriteTo(&buf); err != nil {
		return err
	}

	*b = buf
	return nil
}

// Extract the topic from an io.Reader.
func Extract(p io.Reader) (Topic, error) {
	var cmdBuf [1]byte
	if _, err := p.Read(cmdBuf[:]); err != nil {
		return Reject, err
	}

	return Topic(cmdBuf[0]), nil
}

// Write a topic to a Writer.
func Write(r io.Writer, topic Topic) error {
	_, err := r.Write([]byte{byte(topic)})
	return err
}
