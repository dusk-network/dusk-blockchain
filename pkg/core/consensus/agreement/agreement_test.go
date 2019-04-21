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
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
	"golang.org/x/crypto/ed25519"
)

func TestInitBroker(t *testing.T) {
	committeeMock := mockCommittee(2, true)
	bus := wire.NewEventBus()
	roundChan := consensus.InitRoundUpdate(bus)

	LaunchAgreement(bus, committeeMock, 1)

	round := <-roundChan
	assert.Equal(t, uint64(1), round)
}

func TestBroker(t *testing.T) {
	committeeMock := mockCommittee(2, true)
	_, broker, roundChan := initAgreement(committeeMock)

	hash, _ := crypto.RandEntropy(32)
	e1 := mockAgreementBuf(hash, 1, 1, 2)
	e2 := mockAgreementBuf(hash, 1, 1, 2)
	_ = broker.filter.Collect(e1)
	_ = broker.filter.Collect(e2)

	round := <-roundChan
	assert.Equal(t, uint64(2), round)
}

func TestNoQuorum(t *testing.T) {
	committeeMock := mockCommittee(3, true)
	_, broker, roundChan := initAgreement(committeeMock)
	hash, _ := crypto.RandEntropy(32)
	_ = broker.filter.Collect(mockAgreementBuf(hash, 1, 1, 3))
	_ = broker.filter.Collect(mockAgreementBuf(hash, 1, 1, 3))

	select {
	case <-roundChan:
		assert.FailNow(t, "not supposed to get a round update without reaching quorum")
	case <-time.After(100 * time.Millisecond):
		// all good
	}

	_ = broker.filter.Collect(mockAgreementBuf(hash, 1, 1, 3))
	round := <-roundChan
	assert.Equal(t, uint64(2), round)
}

func TestSkipNoMember(t *testing.T) {
	committeeMock := mockCommittee(1, false)
	_, broker, roundChan := initAgreement(committeeMock)
	hash, _ := crypto.RandEntropy(32)
	_ = broker.filter.Collect(mockAgreementBuf(hash, 1, 1, 1))

	select {
	case <-roundChan:
		assert.FailNow(t, "not supposed to get a round update without reaching quorum")
	case <-time.After(100 * time.Millisecond):
		// all good
	}
}

func TestAgreement(t *testing.T) {
	committeeMock := mockCommittee(1, true)
	bus, _, roundChan := initAgreement(committeeMock)
	hash, _ := crypto.RandEntropy(32)
	bus.Publish(string(topics.Agreement), marshalOutgoing(mockAgreementBuf(hash, 1, 1, 1)))

	round := <-roundChan
	assert.Equal(t, uint64(2), round)

	bus.Publish(string(topics.Agreement), marshalOutgoing(mockAgreementBuf(hash, 1, 1, 1)))

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

func marshalOutgoing(buf *bytes.Buffer) *bytes.Buffer {
	msgBytes := buf.Bytes()
	keys, _ := user.NewRandKeys()
	signature := ed25519.Sign(*keys.EdSecretKey, msgBytes)

	if err := msg.VerifyEd25519Signature(keys.EdPubKeyBytes(), msgBytes, signature); err != nil {
		panic(err)
	}

	b := make([]byte, 0)
	outgoing := bytes.NewBuffer(b)
	// outgoing, _ = wire.AddTopic(outgoing, topics.Agreement)
	_ = encoding.Write512(outgoing, signature)
	_ = encoding.Write256(outgoing, keys.EdPubKeyBytes())
	if _, err := outgoing.Write(msgBytes); err != nil {
		panic(err)
	}
	return outgoing
}

func mockCommittee(quorum int, isMember bool) *mocks.Committee {
	committeeMock := &mocks.Committee{}
	committeeMock.On("Quorum").Return(quorum)
	committeeMock.On("IsMember",
		mock.AnythingOfType("[]uint8"),
		mock.AnythingOfType("uint64"),
		mock.AnythingOfType("uint8")).Return(isMember)
	return committeeMock
}
