package reduction

import (
	"bytes"
	"encoding/binary"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

type (
	unMarshaller struct {
		*consensus.EventHeaderUnmarshaller
		*consensus.EventHeaderMarshaller
	}

	handler interface {
		consensus.EventHandler
		EmbedVoteHash(wire.Event, *bytes.Buffer) error
		MarshalHeader(r *bytes.Buffer, state *consensusState) error
		MarshalVoteSet(r *bytes.Buffer, evs []wire.Event) error
	}
)

func newUnMarshaller() *unMarshaller {
	return &unMarshaller{
		EventHeaderUnmarshaller: consensus.NewEventHeaderUnmarshaller(),
		EventHeaderMarshaller:   &consensus.EventHeaderMarshaller{},
	}
}

func (a *unMarshaller) MarshalVoteSet(r *bytes.Buffer, evs []wire.Event) error {
	if err := encoding.WriteVarInt(r, uint64(len(evs))); err != nil {
		return err
	}

	for _, event := range evs {
		if err := a.Marshal(r, event); err != nil {
			return err
		}
	}

	return nil
}

func (a *unMarshaller) MarshalHeader(r *bytes.Buffer, state *consensusState) error {
	buffer := new(bytes.Buffer)
	// Decoding Round
	if err := encoding.ReadUint64(r, binary.LittleEndian, &state.Round); err != nil {
		return err
	}

	// Decoding Step
	if err := encoding.ReadUint8(r, &state.Step); err != nil {
		return err
	}

	if _, err := buffer.Write(r.Bytes()); err != nil {
		return err
	}
	return nil
}
