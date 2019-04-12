package committee

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/prerror"
)

type (
	// Committee is the interface for operations depending on the set of Provisioners extracted for a fiven step
	Committee interface {
		wire.EventPrioritizer
		// isMember can accept a BLS Public Key or an Ed25519
		IsMember([]byte) bool
		GetVotingCommittee(uint64, uint8) (map[string]uint8, error)
		VerifyVoteSet(voteSet []wire.Event, hash []byte, round uint64, step uint8) *prerror.PrError
		Quorum() int
	}

	// NotaryEvent is the message that encapsulates data relevant for components relying on committee information
	NotaryEvent struct {
		*consensus.EventHeader
		VoteSet       []wire.Event
		SignedVoteSet []byte
		AgreedHash    []byte
	}

	// ReductionEvent is a basic reduction event.
	ReductionEvent struct {
		*consensus.EventHeader
		VotedHash  []byte
		SignedHash []byte
	}

	EventUnMarshaller struct {
		*consensus.EventHeaderMarshaller
		*consensus.EventHeaderUnmarshaller
	}

	ReductionEventUnMarshaller struct {
		*EventUnMarshaller
	}

	ReductionUnmarshaller interface {
		wire.EventMarshaller
		wire.EventUnmarshaller
		MarshalVoteSet(*bytes.Buffer, []wire.Event) error
		UnmarshalVoteSet(*bytes.Buffer) ([]wire.Event, error)
	}

	// NotaryEventUnMarshaller implements both Marshaller and Unmarshaller interface
	NotaryEventUnMarshaller struct {
		*EventUnMarshaller
		ReductionUnmarshaller
	}

	// Collector is a helper that groups common operations performed on Events related to a committee
	Collector struct {
		*consensus.StepEventAccumulator
		Committee    Committee
		CurrentRound uint64

		// TODO: review
		RepropagationChannel chan *bytes.Buffer
	}

	// Selector is basically a picker of Events based on the priority of their sender
	Selector struct {
		EventChan     chan wire.Event
		BestEventChan chan wire.Event
		StopChan      chan bool
		committee     Committee
	}
)

// NewNotaryEvent creates an empty Event
func NewNotaryEvent() *NotaryEvent {
	return &NotaryEvent{
		EventHeader: &consensus.EventHeader{},
	}
}

// Equal as specified in the Event interface
func (e *ReductionEvent) Equal(ev wire.Event) bool {
	other, ok := ev.(*ReductionEvent)
	return ok && (bytes.Equal(e.PubKeyBLS, other.PubKeyBLS)) &&
		(e.Round == other.Round) && (e.Step == other.Step)
}

// Equal as specified in the Event interface
func (ceh *NotaryEvent) Equal(e wire.Event) bool {
	other, ok := e.(*NotaryEvent)
	return ok && ceh.EventHeader.Equal(other) && bytes.Equal(other.SignedVoteSet, ceh.SignedVoteSet)
}

func NewEventUnMarshaller(validate func([]byte, []byte, []byte) error) *EventUnMarshaller {
	return &EventUnMarshaller{
		EventHeaderMarshaller:   new(consensus.EventHeaderMarshaller),
		EventHeaderUnmarshaller: consensus.NewEventHeaderUnmarshaller(validate),
	}
}

func NewReductionEventUnMarshaller(validate func([]byte, []byte, []byte) error) *ReductionEventUnMarshaller {
	return &ReductionEventUnMarshaller{NewEventUnMarshaller(validate)}
}

// NewNotaryEventUnMarshaller creates a new NotaryEventUnMarshaller. Internally it creates an EventHeaderUnMarshaller which takes care of Decoding and Encoding operations
func NewNotaryEventUnMarshaller(validate func([]byte, []byte, []byte) error) *NotaryEventUnMarshaller {

	return &NotaryEventUnMarshaller{
		ReductionUnmarshaller: NewReductionEventUnMarshaller(func([]byte, []byte, []byte) error { return nil }),
		EventUnMarshaller:     NewEventUnMarshaller(validate),
	}
}

// Unmarshal unmarshals the buffer into a CommitteeEvent
func (a *ReductionEventUnMarshaller) Unmarshal(r *bytes.Buffer, ev wire.Event) error {
	bev := ev.(*ReductionEvent)
	if err := a.EventHeaderUnmarshaller.Unmarshal(r, bev.EventHeader); err != nil {
		return err
	}

	if err := encoding.Read256(r, &bev.VotedHash); err != nil {
		return err
	}

	if err := encoding.ReadBLS(r, &bev.SignedHash); err != nil {
		return err
	}

	return nil
}

// Marshal a ReductionEvent into a buffer.
func (a *ReductionEventUnMarshaller) Marshal(r *bytes.Buffer, ev wire.Event) error {
	bev := ev.(*ReductionEvent)
	if err := a.EventHeaderMarshaller.Marshal(r, bev.EventHeader); err != nil {
		return err
	}

	if err := encoding.Write256(r, bev.VotedHash); err != nil {
		return err
	}

	if err := encoding.WriteBLS(r, bev.SignedHash); err != nil {
		return err
	}

	return nil
}

func (a *ReductionEventUnMarshaller) UnmarshalVoteSet(r *bytes.Buffer) ([]wire.Event, error) {
	length, err := encoding.ReadVarInt(r)
	if err != nil {
		return nil, err
	}

	evs := make([]wire.Event, length)
	for i := uint64(0); i < length; i++ {
		rev := &ReductionEvent{
			EventHeader: &consensus.EventHeader{},
		}
		if err := a.Unmarshal(r, rev); err != nil {
			return nil, err
		}

		evs[i] = rev
	}

	return evs, nil
}

func (a *ReductionEventUnMarshaller) MarshalVoteSet(r *bytes.Buffer, evs []wire.Event) error {
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

// Unmarshal unmarshals the buffer into a CommitteeEventHeader
// Field order is the following:
// * Consensus Header [BLS Public Key; Round; Step]
// * Committee Header [Signed Vote Set; Vote Set; BlockHash]
func (ceu *NotaryEventUnMarshaller) Unmarshal(r *bytes.Buffer, ev wire.Event) error {
	// check if the buffer has contents first
	// if not, we did not get any messages this round
	// TODO: review this
	if r.Len() == 0 {
		return nil
	}

	cev := ev.(*NotaryEvent)
	if err := ceu.EventHeaderUnmarshaller.Unmarshal(r, cev.EventHeader); err != nil {
		return err
	}

	if err := encoding.ReadBLS(r, &cev.SignedVoteSet); err != nil {
		return err
	}

	voteSet, err := ceu.UnmarshalVoteSet(r)
	if err != nil {
		return err
	}
	cev.VoteSet = voteSet

	if err := encoding.Read256(r, &cev.AgreedHash); err != nil {
		return err
	}

	return nil
}

// Marshal the buffer into a committee Event
// Field order is the following:
// * Consensus Header [BLS Public Key; Round; Step]
// * Committee Header [Signed Vote Set; Vote Set; BlockHash]
func (ceu *NotaryEventUnMarshaller) Marshal(r *bytes.Buffer, ev wire.Event) error {
	// TODO: review
	cev, ok := ev.(*NotaryEvent)
	if !ok {
		// cev is nil
		return nil
	}

	if err := ceu.EventHeaderMarshaller.Marshal(r, cev.EventHeader); err != nil {
		return err
	}

	// Marshal BLS Signature of VoteSet
	if err := encoding.WriteBLS(r, cev.SignedVoteSet); err != nil {
		return err
	}

	// Marshal VoteSet
	if err := ceu.MarshalVoteSet(r, cev.VoteSet); err != nil {
		return err
	}

	if err := encoding.Write256(r, cev.AgreedHash); err != nil {
		return err
	}
	// TODO: write the vote set to the buffer
	return nil
}

//ShouldBeSkipped is a shortcut for validating if an Event is relevant
// NOTE: currentRound is handled by some other process, so it is not this component's responsibility to handle corner cases (for example being on an obsolete round because of a disconnect, etc)
// Deprecated: Collectors should use Collector.ShouldSkip instead, considering that verification of Events should be decoupled from syntactic validation and the decision flow should likely be handled differently by different components
func (cc *Collector) ShouldBeSkipped(m *NotaryEvent) bool {
	shouldSkip := cc.ShouldSkip(m, m.Round, m.Step)
	//TODO: the round element needs to be reassessed
	err := cc.Committee.VerifyVoteSet(m.VoteSet, m.AgreedHash, m.Round, m.Step)
	failedVerification := err != nil
	return shouldSkip || failedVerification
}

// ShouldSkip checks if the message is not propagated by a committee member, that is not a duplicate (and in this case should probably check if the Provisioner is malicious) and that is relevant to the current round
func (cc *Collector) ShouldSkip(ev wire.Event, round uint64, step uint8) bool {
	isDupe := cc.Contains(ev, string(step))
	isPleb := !cc.Committee.IsMember(ev.Sender())
	return isDupe || isPleb
}

// UpdateRound is a utility function that can be overridden by the embedding collector in case of custom behaviour when updating the current round
func (cc *Collector) UpdateRound(round uint64) {
	cc.CurrentRound = round
}
