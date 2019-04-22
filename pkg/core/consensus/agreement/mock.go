package agreement

import (
	"bytes"

	"github.com/stretchr/testify/mock"
	"gitlab.dusk.network/dusk-core/dusk-go/mocks"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/events"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/voting"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"golang.org/x/crypto/ed25519"
)

// PublishMock is a mock-up method to facilitate testing of publishing of Agreement events
func PublishMock(bus wire.EventBroker, hash []byte, round uint64, step uint8, voteNr int) {
	buf := MockAgreementBuf(hash, round, step, voteNr)
	// buf = MarshalOutgoing(buf)
	bus.Publish(msg.OutgoingBlockAgreementTopic, buf)
}

func MockAgreement(keys *user.Keys, hash []byte, round uint64, step uint8, votes []wire.Event) *events.Agreement {
	var err error
	ev := events.NewAgreement()
	ev.Round = round
	ev.Step = step
	ev.PubKeyBLS = keys.BLSPubKey.Marshal()
	signer := voting.NewAgreementSigner(keys)
	ev.VoteSet = votes
	ev.AgreedHash = hash
	ev.SignedVoteSet, err = signer.SignVotes(ev.VoteSet)
	if err != nil {
		panic(err)
	}
	return ev
}

func MockVote(keys *user.Keys, hash []byte, round uint64, step uint8) (*user.Keys, wire.Event) {
	vote := events.NewReduction()
	vote.Header.Round = round
	vote.Header.Step = step

	if keys == nil {
		keys, _ = user.NewRandKeys()
	}

	vote.Header.PubKeyBLS = keys.BLSPubKey.Marshal()
	vote.VotedHash = hash
	sigma, _ := bls.Sign(keys.BLSSecretKey, keys.BLSPubKey, hash)
	vote.SignedHash = sigma.Compress()
	return keys, vote
}

func MockVoteAgreement(hash []byte, round uint64, step uint8, voteNr int) wire.Event {
	var k *user.Keys
	var e wire.Event
	votes := make([]wire.Event, 0)
	for i := 0; i < voteNr; i++ {
		if i == 0 {
			k, e = MockVote(nil, hash, round, step)
			votes = append(votes, e)
			continue
		}
		_, e = MockVote(nil, hash, round, step)
		votes = append(votes, e)
	}
	ev := MockAgreement(k, hash, round, step, votes)
	return ev
}

func MockAgreementBuf(hash []byte, round uint64, step uint8, voteNr int) *bytes.Buffer {
	ev := MockVoteAgreement(hash, round, step, voteNr)
	signer := voting.NewAgreementSigner(nil)
	b := make([]byte, 0)
	buf := bytes.NewBuffer(b)
	_ = signer.Marshal(buf, ev)
	return buf
}

func MarshalOutgoing(buf *bytes.Buffer) *bytes.Buffer {
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
