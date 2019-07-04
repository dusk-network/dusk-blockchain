package topics

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
	Candidate Topic = "candidate"
	Score     Topic = "score"
	Reduction Topic = "blockreduction"
	Agreement Topic = "blockagreement"

	// Peer topics
	Gossip Topic = "gossip"

	// Blockchain topics
	ChainInfo Topic = "chaininfo"

	// RPC topics
	RPCChainInfo Topic = "rpcchaininfo"

	// Error topics
	NotFound Topic = "notfound"
	Reject   Topic = "reject"
)

// TopicToByteArray turns a Topic to a byte array of size 15,
// to prepare it for sending over the wire protocol.
func TopicToByteArray(cmd Topic) [Size]byte {
	bs := [Size]byte{}
	for i := 0; i < len(cmd); i++ {
		bs[i] = cmd[i]
	}

	return bs
}

// ByteArrayToTopic turns a byte array of size 15 into a Topic,
// for populating a received message header.
func ByteArrayToTopic(cmd [Size]byte) Topic {
	buf := []byte{}
	for i := 0; i < Size; i++ {
		if cmd[i] != 0 {
			buf = append(buf, cmd[i])
		}
	}

	return Topic(buf)
}
