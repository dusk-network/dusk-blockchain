package encoding

import (
	"errors"

	"golang.org/x/crypto/sha3"
)

// verifyIDNonce performs sha3 over concatenated id and nonce to ensure last
// byte is zero
func verifyIDNonce(id [16]byte, nonce []byte) error {

	if len(nonce) != 4 {
		return errors.New("invalid nonce length")
	}

	data := make([]byte, 0)
	data = append(data, id[:]...)
	data = append(data, nonce[:]...)

	hash := sha3.Sum256(data)
	if (hash[31]) == 0 {
		return nil
	}
	return errors.New("Id and Nonce are not valid parameters")
}

// MsgTypeToString translates wire message into string name
func MsgTypeToString(t byte) string {

	switch t {
	case PingMsg:
		return "PING"
	case PongMsg:
		return "PONG"
	case FindNodesMsg:
		return "FIND_NODES"
	case NodesMsg:
		return "NODES"
	case BroadcastMsg:
		return "BROADCAST"
	}

	return "UNKNOWN"
}

// ComputeNonce receives the user's `Peer` ID and computes the
// ID nonce in order to be able to join the network.
//
// This operation is basically a PoW algorithm that ensures
// that Sybil attacks are more costly.
func ComputeNonce(id []byte) uint32 {

	var nonce uint32 = 0
	var hash [32]byte

	var nonceBytes [NonceLen]byte
	var data [20]byte
	for {

		byteOrder.PutUint32(nonceBytes[:], nonce)

		copy(data[0:16], id[0:16])
		copy(data[16:20], nonceBytes[0:4])

		hash = sha3.Sum256(data[:])
		if (hash[31]) == 0 {
			return nonce
		}

		// reset buffer
		for i := 0; i < len(nonceBytes); i++ {
			nonceBytes[i] = 0
		}

		nonce++
	}
}
