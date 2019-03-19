package notary

import (
	"bytes"
	"encoding/binary"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// eventHeader is an embeddable struct that groups common operations on notaries
type eventHeader struct {
	*consensus.EventHeader
	VoteSet       []*msg.Vote
	SignedVoteSet []byte
}

// Equal as specified in the Event interface
func (a *eventHeader) Equal(e wire.Event) bool {
	other, ok := e.(*eventHeader)
	return ok && (bytes.Equal(a.PubKeyBLS, other.PubKeyBLS)) && (a.Round == other.Round) && (a.Step == other.Step)
}

type eventHeaderUnmarshaller struct {
	validate func(*bytes.Buffer) error
}

func newEventHeaderUnmarshaller(validate func(*bytes.Buffer) error) *eventHeaderUnmarshaller {
	return &eventUnmarshaller{
		EventHeader: &consensus.NewEventHeader{validate}
	}
}

// Unmarshal unmarshals the buffer into a NotaryEvent
// Field order is the following:
// * Consensus Header [BLS Public Key; Round; Step]
// * Notary Header [Signed Vote Set; Vote Set; Block Hash;]
// * Notary Payload (handled by Notary's specific Unmarshaller)
func (a *eventHeaderUnmarshaller) Unmarshal(r *bytes.Buffer, ev wire.Event) error {
	if err := a.validate(r); err != nil {
		return err
	}

	// if the injection is unsuccessful, panic
	notaryEvent := ev.(*notaryEvent)

	if err := encoding.ReadBLS(r, &notaryEvent.SignedVoteSet); err != nil {
		return err
	}

	voteSet, err := msg.DecodeVoteSet(r)
	if err != nil {
		return err
	}
	notaryEvent.VoteSet = voteSet

	if err := encoding.Read256(r, &notaryEvent.BlockHash); err != nil {
		return err
	}

	return nil
}

type eventHeaderMarshaller struct {}

func (ehm *EventHeaderMarshaller) Marshal(buffer *bytes.Buffer, eh eventHeader) {

	// Marshal Header
	marshaller := *consensus.EventHeaderMarshaller{}
	marshaller.Marshal(buffer, neh.EventHeader)

	// Marshal BLS Signature of VoteSet
	if err := encoding.WriteBLS(buffer, signedBlockHash.Compress()); err != nil {
		return nil, err
	}

	// Marshal VoteSet
	bvotes, err := msg.EncodeVoteSet(neh.Votes)
	if err != nil {
		return nil, err
	}

	if err := encoding.WriteVarBytes(buffer, bvotes); err != nil {
		return nil, err
	}

	// Marshal BlockHash
	if err := encoding.Write256(buffer, neh.BlockHash); err != nil {
		return nil, err
	}

	return buffer, nil
}
