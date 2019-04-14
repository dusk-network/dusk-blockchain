package agreement

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.dusk.network/dusk-core/dusk-go/mocks"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/events"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
	"golang.org/x/crypto/ed25519"
)

func TestInitBroker(t *testing.T) {
	committeeMock := mockCommittee(2, true, nil)
	bus := wire.NewEventBus()
	roundChan := consensus.InitRoundUpdate(bus)

	LaunchAgreement(bus, committeeMock, 1)

	round := <-roundChan
	assert.Equal(t, uint64(1), round)
}

func TestBroker(t *testing.T) {
	committeeMock := mockCommittee(2, true, nil)
	_, broker, roundChan := initAgreement(committeeMock)

	hash, _ := crypto.RandEntropy(32)
	_ = broker.filter.Collect(marshalMsg(hash, 1, 1))
	_ = broker.filter.Collect(marshalMsg(hash, 1, 1))

	round := <-roundChan
	assert.Equal(t, uint64(2), round)
}

func TestNoQuorum(t *testing.T) {
	committeeMock := mockCommittee(3, true, nil)
	_, broker, roundChan := initAgreement(committeeMock)
	hash, _ := crypto.RandEntropy(32)
	_ = broker.filter.Collect(marshalMsg(hash, 1, 1))
	_ = broker.filter.Collect(marshalMsg(hash, 1, 1))

	select {
	case <-roundChan:
		assert.FailNow(t, "not supposed to get a round update without reaching quorum")
	case <-time.After(100 * time.Millisecond):
		// all good
	}

	_ = broker.filter.Collect(marshalMsg(hash, 1, 1))
	round := <-roundChan
	assert.Equal(t, uint64(2), round)
}

func TestSkipNoMember(t *testing.T) {
	committeeMock := mockCommittee(1, false, nil)
	_, broker, roundChan := initAgreement(committeeMock)
	hash, _ := crypto.RandEntropy(32)
	_ = broker.filter.Collect(marshalMsg(hash, 1, 1))
	_ = broker.filter.Collect(marshalMsg(hash, 1, 1))

	select {
	case <-roundChan:
		assert.FailNow(t, "not supposed to get a round update without reaching quorum")
	case <-time.After(100 * time.Millisecond):
		// all good
	}
}

func TestAgreement(t *testing.T) {
	committeeMock := mockCommittee(1, true, nil)
	bus, _, roundChan := initAgreement(committeeMock)
	hash, _ := crypto.RandEntropy(32)
	bus.Publish(string(topics.Agreement), marshalOutgoing(hash, 1, 1))

	round := <-roundChan
	assert.Equal(t, uint64(2), round)

	bus.Publish(string(topics.Agreement), marshalOutgoing(hash, 1, 2))

	select {
	case <-roundChan:
		assert.FailNow(t, "Previous round messages should not trigger a phase update")
	case <-time.After(100 * time.Millisecond):
		// all well
		return
	}
}

func initAgreement(c committee.Committee) (wire.EventBroker, *broker, chan uint64) {
	bus := wire.NewEventBus()
	roundChan := consensus.InitRoundUpdate(bus)
	broker := LaunchAgreement(bus, c, 1)
	// we need to discard the first update since it is triggered directly as it is supposed to update the round to all other consensus compoenents
	<-roundChan
	return bus, broker, roundChan
}

func marshalOutgoing(blockhash []byte, round uint64, step uint8) *bytes.Buffer {
	ev := events.NewAgreement()
	ev.Header.Round = round
	ev.Header.Step = step
	ev.PubKeyBLS, _ = crypto.RandEntropy(32)
	ev.AgreedHash = blockhash
	// we leave VoteSet empty but it doesn't matter as the mocked committee will OK it anyway
	ev.SignedVoteSet, _ = crypto.RandEntropy(33)
	ev.VoteSet = make([]wire.Event, 0)

	m := events.NewAgreementUnMarshaller()
	b := make([]byte, 0)
	buf := bytes.NewBuffer(b)
	_ = m.Marshal(buf, ev)

	msgBytes := buf.Bytes()
	keys, _ := user.NewRandKeys()
	signature := ed25519.Sign(*keys.EdSecretKey, msgBytes)

	if err := msg.VerifyEd25519Signature(keys.EdPubKeyBytes(), msgBytes, signature); err != nil {
		panic(err)
	}

	b = make([]byte, 0)
	outgoing := bytes.NewBuffer(b)
	// outgoing, _ = wire.AddTopic(outgoing, topics.Agreement)
	_ = encoding.Write512(outgoing, signature)
	_ = encoding.Write256(outgoing, keys.EdPubKeyBytes())
	_, _ = outgoing.Write(msgBytes)
	return outgoing
}

func marshalMsg(blockhash []byte, round uint64, step uint8) *bytes.Buffer {
	ev := events.NewAgreement()
	ev.Header.Round = round
	ev.Header.Step = step
	ev.PubKeyBLS, _ = crypto.RandEntropy(32)
	ev.AgreedHash = blockhash
	// we leave VoteSet empty klbut it doesn't matter as the mocked committee will OK it anyway
	ev.SignedVoteSet, _ = crypto.RandEntropy(33)
	ev.VoteSet = make([]wire.Event, 0)

	m := events.NewAgreementUnMarshaller()
	b := make([]byte, 0)
	buf := bytes.NewBuffer(b)
	_ = m.Marshal(buf, ev)
	return buf
}

func mockCommittee(quorum int, isMember bool, verification error) *mocks.Committee {
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
