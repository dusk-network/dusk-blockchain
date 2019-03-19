package notary

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.dusk.network/dusk-core/dusk-go/mocks"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

func TestNotaryEventHeaderUnmarshaller(t *testing.T) {
	validateFunc := func(*bytes.Buffer) error {
		return nil
	}
	unmarshaller := newNotaryEventHeaderUnmarshaller(validateFunc)

	step := uint8(1)
	round := uint64(120)
	blockHash, err := crypto.RandEntropy(32)
	assert.Empty(t, err)
	buf, err := mockNotaryEventBuffer(blockHash, round, step)
	assert.Empty(t, err)

	ev := &notaryEvent{}
	assert.Empty(t, unmarshaller.Unmarshal(buf, ev))
	assert.Equal(t, ev.Step, step)
	assert.Equal(t, ev.Round, round)
	assert.Equal(t, ev.BlockHash, blockHash)

	assert.NotEmpty(t, ev.VoteSet)
	assert.NotEmpty(t, ev.SignedVoteSet)
	assert.NotEmpty(t, ev.PubKeyBLS)
}

func mockNotaryEventBuffer(blockHash []byte, round uint64, step uint8) (*bytes.Buffer, error) {
	pub, priv, _ := bls.GenKeyPair(rand.Reader)
	byte32, err := crypto.RandEntropy(32)
	if err != nil {
		return nil, err
	}
	signedVote, err := bls.Sign(privKeyBLS, pubKeyBLS, byte32)
	if err != nil {
		return nil, err
	}

	vote := newVote(byte32, pubKeyBLS.Marshal(), signedVote.Compress(), step)

	eh := *eventHeader{
		EventHeader: &consensus.EventHeader{
			Round: round,
			Step: step,
			PubKeyBLS: pub.Marshal(),
		}
		SignedVoteSet: signedVote,
		VoteSet: []*msg.Vote{vote},
		BlockHash: byte32,
	}

	ehm := &eventHeaderMarshaller{}
	buf := new(bytes.Buffer)
	return ehm.Marshal(buf, eh), nil
}

func newVote(hash []byte, pub []byte, sig []byte, step uint8) *msg.Vote {
	return &msg.Vote{
		VotedHash:  hash,
		PubKeyBLS:  pub,
		SignedHash: sig,
		Step:       step,
	}
}

func mockCommittee(quorum int, isMember bool, verification error) user.Committee {
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
