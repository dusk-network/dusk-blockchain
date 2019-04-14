package events_test

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/events"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

func TestAgreementEventUnmarshal(t *testing.T) {
	unmarshaller := c.NewAgreementEventUnMarshaller(validate)

	step := uint8(1)
	round := uint64(120)
	blockHash, err := crypto.RandEntropy(32)
	assert.Empty(t, err)
	buf, err := mockEventBuffer(blockHash, round, step)
	assert.Empty(t, err)

	ev := c.NewAgreementEvent()
	assert.Empty(t, unmarshaller.Unmarshal(buf, ev))
	assert.Equal(t, ev.Step, step)
	assert.Equal(t, ev.Round, round)
	assert.Equal(t, ev.AgreedHash, blockHash)

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

	byte64, _ := crypto.RandEntropy(64)
	eh := &c.AgreementEvent{
		EventHeader: &events.EventHeader{
			Signature: byte64,
			PubKeyEd:  byte32,
			Round:     round,
			Step:      step,
			PubKeyBLS: pub.Marshal(),
		},
		SignedVoteSet: signedVote.Compress(),
		VoteSet:       []wire.Event{vote},
		AgreedHash:    blockHash,
	}

	ehm := events.NewAgreementEventUnMarshaller(validate)
	buf := new(bytes.Buffer)
	ehm.Marshal(buf, eh)
	return buf, nil
}

func newReductionEvent(hash []byte, pub []byte, sig []byte, step uint8) *c.ReductionEvent {
	byte32, _ := crypto.RandEntropy(32)
	byte64, _ := crypto.RandEntropy(64)
	return &c.ReductionEvent{
		EventHeader: &events.EventHeader{
			Signature: byte64,
			PubKeyEd:  byte32,
			PubKeyBLS: pub,
			Round:     1,
			Step:      1,
		},
		VotedHash:  hash,
		SignedHash: sig,
	}
}

func validate(bs1, bs2, bs3 []byte) error {
	return nil
}
