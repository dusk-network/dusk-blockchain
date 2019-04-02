package committee_test

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	c "gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

func TestNotaryEventUnmarshal(t *testing.T) {
	unmarshaller := c.NewNotaryEventUnMarshaller(c.NewReductionEventUnMarshaller())

	step := uint8(1)
	round := uint64(120)
	blockHash, err := crypto.RandEntropy(32)
	assert.Empty(t, err)
	buf, err := mockEventBuffer(blockHash, round, step)
	assert.Empty(t, err)

	ev := c.NewNotaryEvent()
	assert.Empty(t, unmarshaller.Unmarshal(buf, ev))
	assert.Equal(t, ev.Step, step)
	assert.Equal(t, ev.Round, round)
	assert.Equal(t, ev.BlockHash, blockHash)

	assert.NotEmpty(t, ev.VoteSet)
	assert.NotEmpty(t, ev.SignedVoteSet)
	assert.NotEmpty(t, ev.PubKeyBLS)
}

func mockEventBuffer(blockHash []byte, round uint64, step uint8) (*bytes.Buffer, error) {
	pub, priv, _ := bls.GenKeyPair(rand.Reader)
	byte32, err := crypto.RandEntropy(32)
	if err != nil {
		return nil, err
	}
	signedVote, err := bls.Sign(priv, pub, byte32)
	if err != nil {
		return nil, err
	}

	vote := newReductionEvent(byte32, pub.Marshal(), signedVote.Compress(), step)

	eh := &c.NotaryEvent{
		EventHeader: &consensus.EventHeader{
			Round:     round,
			Step:      step,
			PubKeyBLS: pub.Marshal(),
		},
		SignedVoteSet: signedVote.Compress(),
		VoteSet:       []wire.Event{vote},
		BlockHash:     blockHash,
	}

	ehm := c.NewNotaryEventUnMarshaller(c.NewReductionEventUnMarshaller())
	buf := new(bytes.Buffer)
	ehm.Marshal(buf, eh)
	return buf, nil
}

func newReductionEvent(hash []byte, pub []byte, sig []byte, step uint8) *c.ReductionEvent {
	return &c.ReductionEvent{
		EventHeader: &consensus.EventHeader{
			PubKeyBLS: pub,
			Round:     1,
			Step:      1,
		},
		VotedHash:  hash,
		SignedHash: sig,
	}
}
