package peer

import (
	"bytes"
	"encoding/binary"
	"io"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

// VersionMessage is a version message on the dusk wire protocol.
type VersionMessage struct {
	Version   *protocol.Version
	Timestamp int64
	Services  protocol.ServiceFlag
}

func newVersionMessageBuffer(v *protocol.Version, services protocol.ServiceFlag) (*bytes.Buffer, error) {
	buffer := new(bytes.Buffer)
	if err := v.Encode(buffer); err != nil {
		return nil, err
	}

	if err := encoding.WriteUint64(buffer, binary.LittleEndian,
		uint64(time.Now().Unix())); err != nil {
		return nil, err
	}

	if err := encoding.WriteUint64(buffer, binary.LittleEndian, uint64(services)); err != nil {
		return nil, err
	}

	return buffer, nil
}

func decodeVersionMessage(r io.Reader) (*VersionMessage, error) {
	versionMessage := &VersionMessage{
		Version: &protocol.Version{},
	}
	if err := versionMessage.Version.Decode(r); err != nil {
		return nil, err
	}

	var time uint64
	if err := encoding.ReadUint64(r, binary.LittleEndian, &time); err != nil {
		return nil, err
	}

	versionMessage.Timestamp = int64(time)

	var services uint64
	if err := encoding.ReadUint64(r, binary.LittleEndian, &services); err != nil {
		return nil, err
	}

	versionMessage.Services = protocol.ServiceFlag(services)
	return versionMessage, nil
}
