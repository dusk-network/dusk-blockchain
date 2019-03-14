package topics

// Topic defines a command
type Topic string

// Size is the size of a command field in bytes
const Size = 15

// A list of all valid protocol commands
const (
	// Standard commands
	Version Topic = "version"
	VerAck  Topic = "verack"
	Ping    Topic = "ping"
	Pong    Topic = "pong"

	// Data exchange commands
	Addr           Topic = "addr"
	GetAddr        Topic = "getaddr"
	GetData        Topic = "getdata"
	GetBlocks      Topic = "getblocks"
	GetHeaders     Topic = "getheaders"
	Tx             Topic = "tx"
	Block          Topic = "block"
	Headers        Topic = "headers"
	MemPool        Topic = "mempool"
	Inv            Topic = "inv"
	CertificateReq Topic = "certificatereq"
	Certificate    Topic = "certificate"

	// Consensus commands
	Candidate       Topic = "candidate"
	Score           Topic = "score"
	SigSet          Topic = "sigset"
	BlockReduction  Topic = "blockreduction"
	SigSetReduction Topic = "sigsetreduction"
	Agreement       Topic = "agreement"

	// Error commands
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
