package encoding

import (
	"bytes"
	"encoding/binary"
	"errors"
)

const (
	// Message types over UDP

	PingMsg      = 0
	PongMsg      = 1
	FindNodesMsg = 2
	NodesMsg     = 3

	// Message types over TCP

	BroadcastMsg = 10

	IDLen    = 16
	NonceLen = 4
)

var (
	byteOrder = binary.LittleEndian
)

// BinaryMarshaler interface for marshal/unmarshal wire unit
type BinaryMarshaler interface {

	// Marshal payload to binary
	MarshalBinary(buf *bytes.Buffer) error

	// Unmarshal payload from binary
	UnmarshalBinary(buf *bytes.Buffer) error
}

// MarshalBinary marshals message into binary buffer
func MarshalBinary(header Header, payload BinaryMarshaler, buf *bytes.Buffer) error {

	// marshal header
	if err := header.MarshalBinary(buf); err != nil {
		return err
	}

	// marshal payload, if provided
	if payload != nil {
		if err := payload.MarshalBinary(buf); err != nil {
			return err
		}
	} else {
		// Ensure payload is provided
		switch header.MsgType {
		case PingMsg, PongMsg, FindNodesMsg:
			return nil
		default:
			return errors.New("missing message payload")
		}
	}

	return nil
}
