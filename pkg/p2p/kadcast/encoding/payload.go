package encoding

import (
	"bytes"
	"errors"
)

// BroadcastPayload payload data of BROADCAST message
type BroadcastPayload struct {
	Height      byte
	GossipFrame []byte
}

// NodesPayload payload data of NODES message
type NodesPayload struct {
	Peers []PeerInfo
}

// MarshalBinary implements BinaryMarshaler
func (payload *NodesPayload) MarshalBinary(buf *bytes.Buffer) error {

	peersNum := uint16(len(payload.Peers))
	if peersNum == 0 {
		return errors.New("invalid peers count")
	}

	numBytes := make([]byte, 2)
	byteOrder.PutUint16(numBytes, peersNum)

	if _, err := buf.Write(numBytes); err != nil {
		return err
	}

	for _, p := range payload.Peers {
		if err := p.MarshalBinary(buf); err != nil {
			return err
		}
	}

	return nil
}

// UnmarshalBinary implements BinaryMarshaler
func (payload *NodesPayload) UnmarshalBinary(buf *bytes.Buffer) error {

	var b [2]byte
	if _, err := buf.Read(b[:]); err != nil {
		return err
	}
	num := byteOrder.Uint16(b[:])

	for i := uint16(0); i < num; i++ {
		pinfo := PeerInfo{}
		if err := pinfo.UnmarshalBinary(buf); err != nil {
			return err
		}

		payload.Peers = append(payload.Peers, pinfo)
	}
	return nil
}

// MarshalBinary implements BinaryMarshaler
func (payload *BroadcastPayload) MarshalBinary(buf *bytes.Buffer) error {

	if err := buf.WriteByte(payload.Height); err != nil {
		return err
	}

	lenBytes := make([]byte, 4)
	byteOrder.PutUint32(lenBytes, uint32(len(payload.GossipFrame)))

	if _, err := buf.Write(lenBytes); err != nil {
		return err
	}

	// TODO: Verify length here
	if _, err := buf.Write(payload.GossipFrame); err != nil {
		return err
	}

	return nil
}

// UnmarshalBinary implements BinaryMarshaler
func (payload *BroadcastPayload) UnmarshalBinary(buf *bytes.Buffer) error {

	height, err := buf.ReadByte()
	if err != nil {
		return err
	}

	var b [4]byte
	if _, err := buf.Read(b[:]); err != nil {
		return err
	}
	length := byteOrder.Uint32(b[:])

	gossipFrame := make([]byte, length)
	if _, err := buf.Read(gossipFrame[:]); err != nil {
		return err
	}

	payload.Height = height
	payload.GossipFrame = gossipFrame

	return nil
}
