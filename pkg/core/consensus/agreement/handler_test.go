package agreement

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/events"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/voting"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

func TestVoteVerification(t *testing.T) {
	hash, _ := crypto.RandEntropy(32)
	keys, ev1 := mockVote(nil, hash, 1, 1)
	_, ev2 := mockVote(keys, hash, 1, 2)

	agreement := mockAgreement(keys, hash, 1, 2, []wire.Event{ev1, ev2})

	// mocking voters
	c := mockCommittee(2, true)

	handler := newHandler(c)

	assert.NoError(t, handler.Verify(agreement))
}

func TestDuplicateVoteSetNotAccounted(t *testing.T) {
	hash, _ := crypto.RandEntropy(32)
	k, ev := mockVote(nil, hash, 1, 1)
	_, cpy := mockVote(k, hash, 1, 1)
	agreement := mockAgreement(k, hash, 1, 1, []wire.Event{ev, cpy})

	// mocking voters
	c := mockCommittee(2, true)

	handler := newHandler(c)

	assert.Error(t, handler.Verify(agreement))
}

func mockAgreement(keys *user.Keys, hash []byte, round uint64, step uint8, votes []wire.Event) *events.Agreement {
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

func mockVote(keys *user.Keys, hash []byte, round uint64, step uint8) (*user.Keys, wire.Event) {
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

func mockVoteAgreement(hash []byte, round uint64, step uint8, voteNr int) wire.Event {
	var k *user.Keys
	var e wire.Event
	votes := make([]wire.Event, 0)
	for i := 0; i < voteNr; i++ {
		if i == 0 {
			k, e = mockVote(nil, hash, round, step)
			votes = append(votes, e)
			continue
		}
		_, e = mockVote(nil, hash, round, step)
		votes = append(votes, e)
	}
	ev := mockAgreement(k, hash, round, step, votes)
	return ev
}

func mockAgreementBuf(hash []byte, round uint64, step uint8, voteNr int) *bytes.Buffer {
	ev := mockVoteAgreement(hash, round, step, voteNr)
	signer := voting.NewAgreementSigner(nil)
	b := make([]byte, 0)
	buf := bytes.NewBuffer(b)
	_ = signer.Marshal(buf, ev)
	return buf
}
