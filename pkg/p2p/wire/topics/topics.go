package topics

import (
	"bytes"
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

	// Data exchange topics (RPCBus)
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
	GetRoundResults
	RoundResults
	GetCandidate

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
)

type topicBuf struct {
	Topic
	bytes.Buffer
	str string
}

var Topics = [...]topicBuf{
	topicBuf{Version, *(bytes.NewBuffer([]byte{byte(Version)})), "version"},
	topicBuf{VerAck, *(bytes.NewBuffer([]byte{byte(VerAck)})), "verack"},
	topicBuf{Ping, *(bytes.NewBuffer([]byte{byte(Ping)})), "ping"},
	topicBuf{Pong, *(bytes.NewBuffer([]byte{byte(Pong)})), "pong"},
	topicBuf{Addr, *(bytes.NewBuffer([]byte{byte(Addr)})), "addr"},
	topicBuf{GetAddr, *(bytes.NewBuffer([]byte{byte(GetAddr)})), "getaddr"},
	topicBuf{GetData, *(bytes.NewBuffer([]byte{byte(GetData)})), "getdata"},
	topicBuf{GetBlocks, *(bytes.NewBuffer([]byte{byte(GetBlocks)})), "getblocks"},
	topicBuf{GetHeaders, *(bytes.NewBuffer([]byte{byte(GetHeaders)})), "getheaders"},
	topicBuf{Tx, *(bytes.NewBuffer([]byte{byte(Tx)})), "tx"},
	topicBuf{Block, *(bytes.NewBuffer([]byte{byte(Block)})), "block"},
	topicBuf{AcceptedBlock, *(bytes.NewBuffer([]byte{byte(AcceptedBlock)})), "acceptedblock"},
	topicBuf{Headers, *(bytes.NewBuffer([]byte{byte(Headers)})), "headers"},
	topicBuf{MemPool, *(bytes.NewBuffer([]byte{byte(MemPool)})), "mempool"},
	topicBuf{Inv, *(bytes.NewBuffer([]byte{byte(Inv)})), "inv"},
	topicBuf{Certificate, *(bytes.NewBuffer([]byte{byte(Certificate)})), "certificate"},
	topicBuf{GetRoundResults, *(bytes.NewBuffer([]byte{byte(GetRoundResults)})), "getroundresults"},
	topicBuf{RoundResults, *(bytes.NewBuffer([]byte{byte(RoundResults)})), "roundresults"},
	topicBuf{GetCandidate, *(bytes.NewBuffer([]byte{byte(GetCandidate)})), "getcandidate"},
	topicBuf{Candidate, *(bytes.NewBuffer([]byte{byte(Candidate)})), "candidate"},
	topicBuf{Score, *(bytes.NewBuffer([]byte{byte(Score)})), "score"},
	topicBuf{Reduction, *(bytes.NewBuffer([]byte{byte(Reduction)})), "reduction"},
	topicBuf{Agreement, *(bytes.NewBuffer([]byte{byte(Agreement)})), "agreement"},
	topicBuf{Gossip, *(bytes.NewBuffer([]byte{byte(Gossip)})), "gossip"},
	topicBuf{NotFound, *(bytes.NewBuffer([]byte{byte(NotFound)})), "notfound"},
	topicBuf{Unknown, *(bytes.NewBuffer([]byte{byte(Unknown)})), "unknown"},
	topicBuf{Reject, *(bytes.NewBuffer([]byte{byte(Reject)})), "reject"},
	topicBuf{Initialization, *(bytes.NewBuffer([]byte{byte(Initialization)})), "initialization"},
	topicBuf{RoundUpdate, *(bytes.NewBuffer([]byte{byte(RoundUpdate)})), "roundupdate"},
	topicBuf{BestScore, *(bytes.NewBuffer([]byte{byte(BestScore)})), "bestscore"},
	topicBuf{Quit, *(bytes.NewBuffer([]byte{byte(Quit)})), "quit"},
	topicBuf{Log, *(bytes.NewBuffer([]byte{byte(Log)})), "log"},
	topicBuf{Monitor, *(bytes.NewBuffer([]byte{byte(Log)})), "monitor_topic"},
	topicBuf{Test, *(bytes.NewBuffer([]byte{byte(Test)})), "__test"},
	topicBuf{StepVotes, *(bytes.NewBuffer([]byte{byte(StepVotes)})), "stepvotes"},
	topicBuf{ScoreEvent, *(bytes.NewBuffer([]byte{byte(ScoreEvent)})), "scoreevent"},
	topicBuf{Generation, *(bytes.NewBuffer([]byte{byte(Generation)})), "generation"},
	topicBuf{Restart, *(bytes.NewBuffer([]byte{byte(Restart)})), "restart"},
	topicBuf{StopConsensus, *(bytes.NewBuffer([]byte{byte(StopConsensus)})), "stopconsensus"},
	topicBuf{IntermediateBlock, *(bytes.NewBuffer([]byte{byte(IntermediateBlock)})), "intermediateblock"},
	topicBuf{HighestSeen, *(bytes.NewBuffer([]byte{byte(HighestSeen)})), "highestseen"},
	topicBuf{ValidCandidateHash, *(bytes.NewBuffer([]byte{byte(ValidCandidateHash)})), "validcandidatehash"},
}

func (t Topic) ToBuffer() bytes.Buffer {
	var buf bytes.Buffer
	if len(Topics) < int(t) {
		buf = *(new(bytes.Buffer))
	}

	buf = Topics[int(t)].Buffer
	return buf
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
