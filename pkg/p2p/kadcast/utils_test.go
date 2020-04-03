package kadcast

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
)

func TestPOW(t *testing.T) {
	a := Peer{
		ip:   [4]byte{192, 169, 1, 1},
		port: 25519,
		id:   [16]byte{22, 22, 22, 22, 22, 22, 22, 22, 22, 22, 22, 22, 22, 22, 22, 22},
	}

	println(a.computePeerNonce())
}

func TestWriteTCPFrame(t *testing.T) {
	writePayload := bytes.NewBufferString("some payload data")

	conn := &bytes.Buffer{}
	if err := writeTCPFrame(conn, writePayload.Bytes()); err != nil {
		t.Error(err)
	}

	readPayload, _, err := readTCPFrame(conn)
	if err != nil {
		t.Error(err)
	}

	if !bytes.Equal(readPayload, writePayload.Bytes()) {
		t.Error("payload read is invalid")
	}
}

func TestWriteTCPFrameErr(t *testing.T) {
	writePayload := make([]byte, MaxFrameSize+1)

	conn := &bytes.Buffer{}
	if err := writeTCPFrame(conn, writePayload); err != ErrExceedMaxLen {
		t.Error("expect frame size error")
	}
}

func TestReadTCPFrameErr(t *testing.T) {

	frame := new(bytes.Buffer)
	_ = encoding.WriteUint32LE(frame, MaxFrameSize+1)

	readPayload, _, err := readTCPFrame(frame)
	if readPayload != nil {
		t.Error("expect an nil payload ")
	}
	if err != ErrExceedMaxLen {
		t.Error("expect frame size error")
	}
}
