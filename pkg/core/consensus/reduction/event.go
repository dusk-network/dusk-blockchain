package reduction

import (
	"bytes"
	"encoding/binary"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

type (
	unMarshaller struct {
		*committee.ReductionEventUnMarshaller
	}

	handler interface {
		consensus.EventHandler
		EmbedVoteHash(wire.Event, *bytes.Buffer) error
		MarshalHeader(r *bytes.Buffer, state *consensusState) (*bytes.Buffer, error)
		MarshalVoteSet(r *bytes.Buffer, evs []wire.Event) error
	}
)

func newUnMarshaller(validate func([]byte, []byte, []byte) error) *unMarshaller {
	return &unMarshaller{committee.NewReductionEventUnMarshaller(validate)}
}

func (a *unMarshaller) MarshalHeader(r *bytes.Buffer, state *consensusState) (*bytes.Buffer, error) {
	buffer := new(bytes.Buffer)
	// Decoding Round
	if err := encoding.WriteUint64(buffer, binary.LittleEndian, state.Round); err != nil {
		return nil, err
	}

	// Decoding Step
	if err := encoding.WriteUint8(buffer, state.Step); err != nil {
		return nil, err
	}

	if _, err := buffer.Write(r.Bytes()); err != nil {
		return nil, err
	}
	return buffer, nil
}
