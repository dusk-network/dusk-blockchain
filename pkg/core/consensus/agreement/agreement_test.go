package agreement

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.dusk.network/dusk-core/dusk-go/mocks"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

func TestSimpleCollection(t *testing.T) {
	committeeMock := mockCommittee(2, true, nil)
	bc := newBlockCollector(committeeMock, 1)

	blockHash := []byte("pippo")
	bc.Unmarshaller = mockBEUnmarshaller(blockHash, 1, 1)
	bc.Collect(bytes.NewBuffer([]byte{}))
	bc.Collect(bytes.NewBuffer([]byte{}))

	select {
	case res := <-bc.RoundChan:
		assert.Equal(t, uint64(2), res)
		// testing that we clean after collection
		assert.Equal(t, 0, len(bc.StepEventAccumulator.Map))
	case <-time.After(1 * time.Second):
		assert.Fail(t, "Collection did not complete")
	}
}

func TestNoQuorumCollection(t *testing.T) {
	committeeMock := mockCommittee(3, true, nil)
	bc := newBlockCollector(committeeMock, 1)

	blockHash := []byte("pippo")
	bc.Unmarshaller = mockBEUnmarshaller(blockHash, 1, 1)
	bc.Collect(bytes.NewBuffer([]byte{}))
	bc.Collect(bytes.NewBuffer([]byte{}))

	select {
	case <-bc.RoundChan:
		assert.Fail(t, "Collection was not supposed to complete since Quorum should not be reached")
	case <-time.After(100 * time.Millisecond):
		// testing that we still have collected for 1 step
		assert.Equal(t, 1, len(bc.StepEventAccumulator.Map))
		// testing that we collected 2 messages
		assert.Equal(t, 2, len(bc.StepEventAccumulator.Map[string(1)]))
	}
}

func TestSkipNoMember(t *testing.T) {
	committeeMock := mockCommittee(1, false, nil)
	bc := newBlockCollector(committeeMock, 1)

	blockHash := []byte("pippo")
	bc.Unmarshaller = mockBEUnmarshaller(blockHash, 1, 1)
	bc.Collect(bytes.NewBuffer([]byte{}))

	select {
	case <-bc.RoundChan:
		assert.Fail(t, "Collection was not supposed to complete since Quorum should not be reached")
	case <-time.After(50 * time.Millisecond):
		// test successfull
	}
}

func TestBlockAgreement(t *testing.T) {
	bus := wire.NewEventBus()
	committee := mockCommittee(1, true, nil)
	agreement := LaunchAgreement(bus, committee, 1)
	blockHash := []byte("pippo")
	agreement.blockCollector.Unmarshaller = mockBEUnmarshaller(blockHash, 1, 1)

	blockChan := make(chan *bytes.Buffer)
	bus.Subscribe(msg.RoundUpdateTopic, blockChan)

	// Get round out of the channel, that was put in from initialization
	// It should be 1
	initRound := <-blockChan
	initNum := binary.LittleEndian.Uint64(initRound.Bytes())
	assert.Equal(t, uint64(1), initNum)

	bus.Publish(string(topics.BlockAgreement), bytes.NewBuffer([]byte("test")))

	result := <-blockChan
	resNum := binary.LittleEndian.Uint64(result.Bytes())
	assert.Equal(t, resNum, uint64(2))

	// test that after a round update messages for previous rounds get ignored
	// we need to wait for the round update to be propagated before publishing other round related messages. This is what this timeout is about
	<-time.After(100 * time.Millisecond)

	agreement.blockCollector.Unmarshaller = mockBEUnmarshaller(blockHash, 1, 2)
	bus.Publish(msg.BlockAgreementTopic, bytes.NewBuffer([]byte("test")))

	select {
	case <-blockChan:
		assert.FailNow(t, "Previous round messages should not trigger a phase update")
	case <-time.After(100 * time.Millisecond):
		// all well
		return
	}

}

type MockBEUnmarshaller struct {
	event *BlockEvent
	err   error
}

func (m *MockBEUnmarshaller) Unmarshal(b *bytes.Buffer, e wire.Event) error {
	if m.err != nil {
		return m.err
	}

	blsPub, _ := crypto.RandEntropy(32)
	ev := e.(*BlockEvent)
	ev.Step = m.event.Step
	ev.Round = m.event.Round
	ev.AgreedHash = m.event.AgreedHash
	ev.PubKeyBLS = blsPub
	return nil
}

// Marshal is not used by the mock unmarshaller.
func (m *MockBEUnmarshaller) Marshal(b *bytes.Buffer, e wire.Event) error {
	return nil
}

func mockBEUnmarshaller(blockHash []byte, round uint64, step uint8) wire.EventUnMarshaller {
	ev := committee.NewAgreementEvent()
	ev.AgreedHash = blockHash
	ev.Step = step
	ev.Round = round

	return &MockBEUnmarshaller{
		event: ev,
		err:   nil,
	}
}

func mockCommittee(quorum int, isMember bool, verification error) committee.Committee {
	committeeMock := &mocks.Committee{}
	committeeMock.On("Quorum").Return(quorum)
	committeeMock.On("VerifyVoteSet",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything).Return(verification)
	committeeMock.On("IsMember", mock.AnythingOfType("[]uint8")).Return(isMember)
	return committeeMock
}
