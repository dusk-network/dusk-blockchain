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

func TestCommitteeEventUnmarshaller(t *testing.T) {
	validateFunc := func(*bytes.Buffer) error {
		return nil
	}
	unmarshaller := newCommitteeEventUnmarshaller(validateFunc)

	step := uint8(1)
	round := uint64(120)
	blockHash, err := crypto.RandEntropy(32)
	assert.Empty(t, err)
	buf, err := mockCommitteeEventBuffer(blockHash, round, step)
	assert.Empty(t, err)

	ev := &committeeEvent{}
	assert.Empty(t, unmarshaller.Unmarshal(buf, ev))
	assert.Equal(t, ev.Step, step)
	assert.Equal(t, ev.Round, round)
	assert.Equal(t, ev.BlockHash, blockHash)

	assert.NotEmpty(t, ev.VoteSet)
	assert.NotEmpty(t, ev.SignedVoteSet)
	assert.NotEmpty(t, ev.PubKeyBLS)
}

func mockCommitteeEventBuffer(blockHash []byte, round uint64, step uint8) (*bytes.Buffer, error) {
	pub, priv, _ := bls.GenKeyPair(rand.Reader)

	return newCommitteeEventBuffer(blockHash, round, step, pub, priv)
}

func newCommitteeEventBuffer(blockHash []byte, round uint64, step uint8,
	pubKeyBLS *bls.PublicKey, privKeyBLS *bls.SecretKey) (*bytes.Buffer, error) {

	byte32, err := crypto.RandEntropy(32)
	if err != nil {
		return nil, err
	}
	signedVote, err := bls.Sign(privKeyBLS, pubKeyBLS, byte32)
	if err != nil {
		return nil, err
	}

	vote := newVote(byte32, pubKeyBLS.Marshal(), signedVote.Compress(), step)

	votes := []*msg.Vote{vote}

	bvotes, err := msg.EncodeVoteSet(votes)
	if err != nil {
		return nil, err
	}

	buffer := bytes.NewBuffer(bvotes)

	signedBlockHash, err := bls.Sign(privKeyBLS, pubKeyBLS, blockHash)
	if err != nil {
		return nil, err
	}

	if err := encoding.WriteBLS(buffer, signedBlockHash.Compress()); err != nil {
		return nil, err
	}

	if err := encoding.WriteVarBytes(buffer, pubKeyBLS.Marshal()); err != nil {
		return nil, err
	}

	if err := encoding.Write256(buffer, blockHash); err != nil {
		return nil, err
	}

	if err := encoding.WriteUint64(buffer, binary.LittleEndian, round); err != nil {
		return nil, err
	}

	if err := encoding.WriteUint8(buffer, step); err != nil {
		return nil, err
	}

	return buffer, nil
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
