package notary_test

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.dusk.network/dusk-core/dusk-go/mocks"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	n "gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/notary"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

func TestSimpleBlockCollection(t *testing.T) {
	committeeMock := &mocks.Committee{}
	committeeMock.On("Quorum").Return(2)
	committeeMock.On("VerifyVoteSet",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything).Return(nil)
	committeeMock.On("IsMember", mock.AnythingOfType("[]uint8")).Return(true)

	blockChan := make(chan []byte, 1)
	validateFunc := func(*bytes.Buffer) error {
		return nil
	}

	bc := n.NewBlockCollector(committeeMock, blockChan, validateFunc)
	bc.UpdateRound(1)

	blockHash, err := crypto.RandEntropy(32)
	assert.Empty(t, err)

	ev1, err := mockBlockEvent(blockHash, 1, 1)
	assert.Empty(t, err)
	ev2, err := mockBlockEvent(blockHash, 1, 1)
	assert.Empty(t, err)
	ev3, err := mockBlockEvent(blockHash, 1, 1)
	assert.Empty(t, err)

	bc.Collect(ev1)
	bc.Collect(ev2)
	bc.Collect(ev3)

	res := <-blockChan
	assert.Equal(t, blockHash, res)
}

func mockBlockEvent(blockHash []byte, round uint64, step uint8) (*bytes.Buffer, error) {
	pub, priv, _ := bls.GenKeyPair(rand.Reader)

	return newBlockEvent(blockHash, round, step, pub, priv)
}

func newBlockEvent(blockHash []byte, round uint64, step uint8,
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
