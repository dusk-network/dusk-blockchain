package peer

import (
	"bytes"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
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

	if err := encoding.WriteUint64LE(buffer, uint64(time.Now().Unix())); err != nil {
		return nil, err
	}

	if err := encoding.WriteUint64LE(buffer, uint64(services)); err != nil {
		return nil, err
	}

	return buffer, nil
}

func decodeVersionMessage(r *bytes.Buffer) (*VersionMessage, error) {
	versionMessage := &VersionMessage{
		Version: &protocol.Version{},
	}
	if err := versionMessage.Version.Decode(r); err != nil {
		return nil, err
	}

	var time uint64
	if err := encoding.ReadUint64LE(r, &time); err != nil {
		return nil, err
	}

	versionMessage.Timestamp = int64(time)

	var services uint64
	if err := encoding.ReadUint64LE(r, &services); err != nil {
		return nil, err
	}

	versionMessage.Services = protocol.ServiceFlag(services)
	return versionMessage, nil
}
