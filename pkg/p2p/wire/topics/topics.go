package topics

import (
	"bufio"
	"io"
)

// Topic defines a topic
type Topic string

// Size is the size of a topic field in bytes
const Size = 15

// A list of all valid topics
const (
	// Standard topics
	Version Topic = "version"
	VerAck  Topic = "verack"
	Ping    Topic = "ping"
	Pong    Topic = "pong"

	// Data exchange topics
	Addr          Topic = "addr"
	GetAddr       Topic = "getaddr"
	GetData       Topic = "getdata"
	GetBlocks     Topic = "getblocks"
	GetHeaders    Topic = "getheaders"
	Tx            Topic = "tx"
	Block         Topic = "block"
	AcceptedBlock Topic = "acceptedblock"
	Headers       Topic = "headers"
	MemPool       Topic = "mempool"
	Inv           Topic = "inv"
	Certificate   Topic = "certificate"

	// Consensus topics
	Candidate      Topic = "candidate"
	Score          Topic = "score"
	Reduction      Topic = "blockreduction"
	Agreement      Topic = "blockagreement"
	StartConsensus Topic = "startconsensus"

	// Peer topics
	Gossip Topic = "gossip"

	// Error topics
	NotFound Topic = "notfound"
	Reject   Topic = "reject"
)

// TopicToByteArray turns a Topic to a byte array of size 15,
// to prepare it for sending over the wire protocol.
func TopicToByteArray(cmd Topic) [Size]byte {
	bs := [Size]byte{}
	copy(bs[:], cmd)
	return bs
}

// ByteArrayToTopic turns a byte array of size 15 into a Topic,
// for populating a received message header.
func ByteArrayToTopic(cmd [Size]byte) Topic {
	buf := []byte{}
	for i := 0; i < Size; i++ {
		if cmd[i] == 0 {
			break
		}
		buf = append(buf, cmd[i])
	}
	return Topic(buf)
}

// Extract the topic by reading the first `topic.Size` bytes
func Extract(p io.Reader) (Topic, error) {
	var cmdBuf [Size]byte
	if _, err := p.Read(cmdBuf[:]); err != nil {
		return "", err
	}

	return ByteArrayToTopic(cmdBuf), nil
}

// Peek the topic without advancing the reader
// Deprecated: Peek is several order of magnitude slower than Extract. Even if having to Tee read the whole buffer, Extract should be used
func Peek(p io.Reader) (Topic, error) {
	var cmdBuf [Size]byte
	bf := bufio.NewReader(p)

	topic, err := bf.Peek(Size)
	if err != nil {
		return "", err
	}

	copy(cmdBuf[:], topic)
	return ByteArrayToTopic(cmdBuf), nil
}

func Write(r io.Writer, topic Topic) error {
	topicBytes := TopicToByteArray(topic)
	if _, err := r.Write(topicBytes[:]); err != nil {
		return err
	}

	return nil
}
