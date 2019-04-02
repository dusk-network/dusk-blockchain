package selection

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"

	"gitlab.dusk.network/dusk-core/dusk-go/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

func TestSelection(t *testing.T) {
	eventBus := wire.New()

	// make a block hash
	blockHash, _ := crypto.RandEntropy(32)

	unmarshaller := mockSSEUnmarshaller(blockHash, 1, 1)
	committeeMock := mockCommittee(1, true, nil, unmarshaller.event)
	timeOut := 500 * time.Millisecond

	// Subscribe to SigSetSelectionTopic
	selectionChan := make(chan *bytes.Buffer, 1)
	eventBus.Subscribe(msg.SigSetSelectionTopic, selectionChan)

	selector := LaunchSignatureSelector(committeeMock, eventBus, timeOut)

	// set unmarshaller
	handler := selector.collector.handler.(*SigSetHandler)
	handler.unMarshaller = unmarshaller
	handler.blockHash = blockHash
	initSelector(eventBus, 1)

	// give the selector some time to init
	time.Sleep(100 * time.Millisecond)

	// start selector
	eventBus.Publish(msg.PhaseUpdateTopic, bytes.NewBuffer(blockHash))

	// send message
	eventBus.Publish(string(topics.SigSet), bytes.NewBuffer([]byte("pippo")))

	result := <-selectionChan
	assert.NotNil(t, result)
}

func TestSelectionNoVotes(t *testing.T) {
	eventBus := wire.New()
	timeOut := 100 * time.Millisecond

	// Subscribe to BlockGenerationTopic
	blockGenerationChan := make(chan *bytes.Buffer, 1)
	eventBus.Subscribe(msg.BlockGenerationTopic, blockGenerationChan)

	LaunchScoreSelectionComponent(eventBus, timeOut, user.BidList{})

	// This starts the score selector instantly, as it fires on round update
	initSelector(eventBus, 1)

	m := <-blockGenerationChan
	assert.Nil(t, m)
}

type MockSSEUnmarshaller struct {
	event *SigSetEvent
	err   error
}

func (m *MockSSEUnmarshaller) Unmarshal(b *bytes.Buffer, e wire.Event) error {
	if m.err != nil {
		return m.err
	}

	blsPub, _ := crypto.RandEntropy(32)
	ev := e.(*SigSetEvent)
	ev.Step = m.event.Step
	ev.Round = m.event.Round
	ev.AgreedHash = m.event.AgreedHash
	ev.PubKeyBLS = blsPub
	ev.VoteSet = m.event.VoteSet
	return nil
}

func (m *MockSSEUnmarshaller) Marshal(b *bytes.Buffer, e wire.Event) error {
	// add something to the buffer so the collector will think we got a result
	if _, err := b.Write([]byte("pippo")); err != nil {
		return err
	}

	return nil
}

func mockSSEUnmarshaller(blockHash []byte, round uint64, step uint8) *MockSSEUnmarshaller {
	ev := committee.NewNotaryEvent()
	ev.AgreedHash = blockHash
	ev.Step = step
	ev.Round = round
	ev.VoteSet = append(ev.VoteSet, newReductionEvent(round, step))

	return &MockSSEUnmarshaller{
		event: ev,
		err:   nil,
	}
}

func initSelector(eventBus *wire.EventBus, round uint64) {
	roundBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(roundBytes[:8], round)
	eventBus.Publish(msg.RoundUpdateTopic, bytes.NewBuffer(roundBytes))
}

func newReductionEvent(round uint64, step uint8) *committee.ReductionEvent {
	byte32, _ := crypto.RandEntropy(32)
	byte64, _ := crypto.RandEntropy(64)
	return &committee.ReductionEvent{
		EventHeader: &consensus.EventHeader{
			Signature: byte64,
			PubKeyEd:  byte32,
			PubKeyBLS: byte32,
			Round:     round,
			Step:      step,
		},
		VotedHash:  byte32,
		SignedHash: byte32,
	}
}

func mockCommittee(quorum int, isMember bool, verification error, event wire.Event) committee.Committee {
	committeeMock := &mocks.Committee{}
	committeeMock.On("Quorum").Return(quorum)
	committeeMock.On("VerifyVoteSet",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything).Return(verification)
	committeeMock.On("IsMember", mock.AnythingOfType("[]uint8")).Return(isMember)
	committeeMock.On("Priority", mock.Anything, mock.Anything).Return(event)
	return committeeMock
}
