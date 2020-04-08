package topics

import (
	"bytes"
	"fmt"
	"io"
)

// Topic defines a topic
type Topic uint8

// A list of all valid topics
const (
	// Standard topics
	Version Topic = iota
	VerAck
	Ping
	Pong

	// Data exchange topics
	Addr
	GetAddr
	GetData
	GetBlocks
	GetHeaders
	Tx
	Block
	AcceptedBlock
	Headers
	MemPool
	Inv
	Certificate
	RoundResults

	// Gossiped topics
	Candidate
	Score
	Reduction
	Agreement

	// Peer topics
	Gossip

	// Error topics
	NotFound
	Unknown
	Reject

	//Internal
	Initialization
	RoundUpdate
	BestScore
	Quit
	Log
	Monitor
	Test
	StepVotes
	ScoreEvent
	Generation
	Restart
	StopConsensus
	IntermediateBlock
	HighestSeen
	ValidCandidateHash

	// RPCBus topics
	GetLastBlock
	GetMempoolTxs
	GetMempoolTxsBySize
	VerifyCandidateBlock
	GetLastCertificate
	SendMempoolTx
	ValidateStateTransition
	ExecuteStateTransition

	// Cross-process RPCBus topics
	// Wallet
	GetMempoolView
	CreateWallet
	CreateFromSeed
	LoadWallet
	SendBidTx
	SendStakeTx
	SendStandardTx
	GetBalance
	GetUnconfirmedBalance
	GetAddress
	GetTxHistory
	AutomateConsensusTxs
	GetSyncProgress
	IsWalletLoaded
	RebuildChain
	ClearWalletDatabase
	StartProfile
	StopProfile

	// Cross-network RPCBus topics
	GetRoundResults
	GetCandidate

	// Monitoring topics
	SyncProgress
)

type topicBuf struct {
	Topic
	bytes.Buffer
	str string
}

// Topics represents the associated string and byte representation respectively
// of the Topic objects
// NOTE: this needs to be in the same order in which the topics are declared
var Topics = [...]topicBuf{
	{Version, *(bytes.NewBuffer([]byte{byte(Version)})), "version"},
	{VerAck, *(bytes.NewBuffer([]byte{byte(VerAck)})), "verack"},
	{Ping, *(bytes.NewBuffer([]byte{byte(Ping)})), "ping"},
	{Pong, *(bytes.NewBuffer([]byte{byte(Pong)})), "pong"},
	{Addr, *(bytes.NewBuffer([]byte{byte(Addr)})), "addr"},
	{GetAddr, *(bytes.NewBuffer([]byte{byte(GetAddr)})), "getaddr"},
	{GetData, *(bytes.NewBuffer([]byte{byte(GetData)})), "getdata"},
	{GetBlocks, *(bytes.NewBuffer([]byte{byte(GetBlocks)})), "getblocks"},
	{GetHeaders, *(bytes.NewBuffer([]byte{byte(GetHeaders)})), "getheaders"},
	{Tx, *(bytes.NewBuffer([]byte{byte(Tx)})), "tx"},
	{Block, *(bytes.NewBuffer([]byte{byte(Block)})), "block"},
	{AcceptedBlock, *(bytes.NewBuffer([]byte{byte(AcceptedBlock)})), "acceptedblock"},
	{Headers, *(bytes.NewBuffer([]byte{byte(Headers)})), "headers"},
	{MemPool, *(bytes.NewBuffer([]byte{byte(MemPool)})), "mempool"},
	{Inv, *(bytes.NewBuffer([]byte{byte(Inv)})), "inv"},
	{Certificate, *(bytes.NewBuffer([]byte{byte(Certificate)})), "certificate"},
	{RoundResults, *(bytes.NewBuffer([]byte{byte(RoundResults)})), "roundresults"},
	{Candidate, *(bytes.NewBuffer([]byte{byte(Candidate)})), "candidate"},
	{Score, *(bytes.NewBuffer([]byte{byte(Score)})), "score"},
	{Reduction, *(bytes.NewBuffer([]byte{byte(Reduction)})), "reduction"},
	{Agreement, *(bytes.NewBuffer([]byte{byte(Agreement)})), "agreement"},
	{Gossip, *(bytes.NewBuffer([]byte{byte(Gossip)})), "gossip"},
	{NotFound, *(bytes.NewBuffer([]byte{byte(NotFound)})), "notfound"},
	{Unknown, *(bytes.NewBuffer([]byte{byte(Unknown)})), "unknown"},
	{Reject, *(bytes.NewBuffer([]byte{byte(Reject)})), "reject"},
	{Initialization, *(bytes.NewBuffer([]byte{byte(Initialization)})), "initialization"},
	{RoundUpdate, *(bytes.NewBuffer([]byte{byte(RoundUpdate)})), "roundupdate"},
	{BestScore, *(bytes.NewBuffer([]byte{byte(BestScore)})), "bestscore"},
	{Quit, *(bytes.NewBuffer([]byte{byte(Quit)})), "quit"},
	{Log, *(bytes.NewBuffer([]byte{byte(Log)})), "log"},
	{Monitor, *(bytes.NewBuffer([]byte{byte(Log)})), "monitor_topic"},
	{Test, *(bytes.NewBuffer([]byte{byte(Test)})), "__test"},
	{StepVotes, *(bytes.NewBuffer([]byte{byte(StepVotes)})), "stepvotes"},
	{ScoreEvent, *(bytes.NewBuffer([]byte{byte(ScoreEvent)})), "scoreevent"},
	{Generation, *(bytes.NewBuffer([]byte{byte(Generation)})), "generation"},
	{Restart, *(bytes.NewBuffer([]byte{byte(Restart)})), "restart"},
	{StopConsensus, *(bytes.NewBuffer([]byte{byte(StopConsensus)})), "stopconsensus"},
	{IntermediateBlock, *(bytes.NewBuffer([]byte{byte(IntermediateBlock)})), "intermediateblock"},
	{HighestSeen, *(bytes.NewBuffer([]byte{byte(HighestSeen)})), "highestseen"},
	{ValidCandidateHash, *(bytes.NewBuffer([]byte{byte(ValidCandidateHash)})), "validcandidatehash"},
	{GetLastBlock, *(bytes.NewBuffer([]byte{byte(GetLastBlock)})), "getlastblock"},
	{GetMempoolTxs, *(bytes.NewBuffer([]byte{byte(GetMempoolTxs)})), "getmempooltxs"},
	{GetMempoolTxsBySize, *(bytes.NewBuffer([]byte{byte(GetMempoolTxsBySize)})), "getmempooltxsbysize"},
	{VerifyCandidateBlock, *(bytes.NewBuffer([]byte{byte(VerifyCandidateBlock)})), "verifycandidateblock"},
	{GetLastCertificate, *(bytes.NewBuffer([]byte{byte(GetLastCertificate)})), "getlastcertificate"},
	{SendMempoolTx, *(bytes.NewBuffer([]byte{byte(SendMempoolTx)})), "sendmempooltx"},
	{ValidateStateTransition, *(bytes.NewBuffer([]byte{byte(ValidateStateTransition)})), "validatestatetransition"},
	{ExecuteStateTransition, *(bytes.NewBuffer([]byte{byte(ExecuteStateTransition)})), "executestatetransition"},
	{GetMempoolView, *(bytes.NewBuffer([]byte{byte(GetMempoolView)})), "getmempoolview"},
	{CreateWallet, *(bytes.NewBuffer([]byte{byte(CreateWallet)})), "createwallet"},
	{CreateFromSeed, *(bytes.NewBuffer([]byte{byte(CreateFromSeed)})), "createfromseed"},
	{LoadWallet, *(bytes.NewBuffer([]byte{byte(LoadWallet)})), "loadwallet"},
	{SendBidTx, *(bytes.NewBuffer([]byte{byte(SendBidTx)})), "sendbidtx"},
	{SendStakeTx, *(bytes.NewBuffer([]byte{byte(SendStakeTx)})), "sendstaketx"},
	{SendStandardTx, *(bytes.NewBuffer([]byte{byte(SendStandardTx)})), "sendstandardtx"},
	{GetBalance, *(bytes.NewBuffer([]byte{byte(GetBalance)})), "getbalance"},
	{GetUnconfirmedBalance, *(bytes.NewBuffer([]byte{byte(GetUnconfirmedBalance)})), "getunconfirmedbalance"},
	{GetAddress, *(bytes.NewBuffer([]byte{byte(GetAddress)})), "getaddress"},
	{GetTxHistory, *(bytes.NewBuffer([]byte{byte(GetTxHistory)})), "gettxhistory"},
	{AutomateConsensusTxs, *(bytes.NewBuffer([]byte{byte(AutomateConsensusTxs)})), "automateconsensustxs"},
	{GetSyncProgress, *(bytes.NewBuffer([]byte{byte(GetSyncProgress)})), "getsyncprogress"},
	{IsWalletLoaded, *(bytes.NewBuffer([]byte{byte(IsWalletLoaded)})), "iswalletloaded"},
	{RebuildChain, *(bytes.NewBuffer([]byte{byte(RebuildChain)})), "rebuildchain"},
	{ClearWalletDatabase, *(bytes.NewBuffer([]byte{byte(ClearWalletDatabase)})), "clearwalletdatabase"},
	{StartProfile, *(bytes.NewBuffer([]byte{byte(StartProfile)})), "startprofile"},
	{StopProfile, *(bytes.NewBuffer([]byte{byte(StopProfile)})), "stopprofile"},
	{GetRoundResults, *(bytes.NewBuffer([]byte{byte(GetRoundResults)})), "getroundresults"},
	{GetCandidate, *(bytes.NewBuffer([]byte{byte(GetCandidate)})), "getcandidate"},
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

// ToBuffer returns Topic as a Buffer
func (t Topic) ToBuffer() bytes.Buffer {
	return Topics[int(t)].Buffer
}

// String representation of a known topic
func (t Topic) String() string {
	if len(Topics) > int(t) {
		return Topics[t].str
	}
	return "unknown"
}

// StringToTopic turns a string into a Topic if the Topic is in the enum of known topics.
// Return Unknown topic if the string is not coupled with any
func StringToTopic(topic string) Topic {
	for _, t := range Topics {
		if t.Topic.String() == topic {
			return t.Topic
		}
	}
	return Unknown
}

// Prepend a topic to a binary-serialized form of a message
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

// Extract the topic from an io.Reader
func Extract(p io.Reader) (Topic, error) {
	var cmdBuf [1]byte
	if _, err := p.Read(cmdBuf[:]); err != nil {
		return Reject, err
	}
	return Topic(cmdBuf[0]), nil
}

// Write a topic to a Writer
func Write(r io.Writer, topic Topic) error {
	_, err := r.Write([]byte{byte(topic)})
	return err
}
