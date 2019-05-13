package reduction

import (
	"bytes"

	"github.com/stretchr/testify/mock"
	"gitlab.dusk.network/dusk-core/dusk-go/mocks"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/events"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/selection"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"golang.org/x/crypto/ed25519"
)

func MockReduction(keys *user.Keys, hash []byte, round uint64, step uint8) (*user.Keys, *events.Reduction) {
	if keys == nil {
		keys, _ = user.NewRandKeys()
	}

	reduction := events.NewReduction()
	reduction.PubKeyBLS = keys.BLSPubKeyBytes
	reduction.Round = round
	reduction.Step = step
	reduction.BlockHash = hash

	r := new(bytes.Buffer)
	vote := &events.Vote{
		Round:     reduction.Round,
		Step:      reduction.Step,
		BlockHash: reduction.BlockHash,
	}
	_ = events.MarshalSignableVote(r, vote)
	sigma, _ := bls.Sign(keys.BLSSecretKey, keys.BLSPubKey, r.Bytes())
	reduction.SignedHash = sigma.Compress()
	return keys, reduction
}

func mockSelectionEventBuffer(hash []byte) *bytes.Buffer {
	// 32 bytes
	score, _ := crypto.RandEntropy(32)
	// Var Bytes
	proof, _ := crypto.RandEntropy(1477)
	// 32 bytes
	z, _ := crypto.RandEntropy(32)
	// Var Bytes
	bidListSubset, _ := crypto.RandEntropy(32)
	// BLS is 33 bytes
	seed, _ := crypto.RandEntropy(33)
	se := &selection.ScoreEvent{
		Round:         uint64(23),
		Score:         score,
		Proof:         proof,
		Z:             z,
		Seed:          seed,
		BidListSubset: bidListSubset,
		VoteHash:      hash,
	}

	b := make([]byte, 0)
	r := bytes.NewBuffer(b)
	_ = selection.MarshalScoreEvent(r, se)
	return r
}

func mockBlockEventBuffer(round uint64, step uint8, hash []byte) *bytes.Buffer {
	keys, _ := user.NewRandKeys()
	signedHash, _ := bls.Sign(keys.BLSSecretKey, keys.BLSPubKey, hash)
	marshaller := events.NewReductionUnMarshaller()

	bev := &events.Reduction{
		Header: &events.Header{
			PubKeyBLS: keys.BLSPubKeyBytes,
			Round:     round,
			Step:      step,
		},
		BlockHash:  hash,
		SignedHash: signedHash.Compress(),
	}

	buf := new(bytes.Buffer)
	_ = marshaller.Marshal(buf, bev)
	edSig := ed25519.Sign(*keys.EdSecretKey, buf.Bytes())
	completeBuf := bytes.NewBuffer(edSig)
	completeBuf.Write(keys.EdPubKeyBytes)
	completeBuf.Write(buf.Bytes())
	return completeBuf
}

func mockCommittee(quorum int, isMember bool, amMember bool) committee.Committee {
	committeeMock := &mocks.Committee{}
	committeeMock.On("Quorum").Return(quorum)
	committeeMock.On("ReportAbsentees", mock.Anything,
		mock.Anything, mock.Anything).Return(nil)
	committeeMock.On("IsMember",
		mock.AnythingOfType("[]uint8"),
		mock.AnythingOfType("uint64"),
		mock.AnythingOfType("uint8")).Return(isMember)
	committeeMock.On("AmMember",
		mock.AnythingOfType("uint64"),
		mock.AnythingOfType("uint8")).Return(amMember)
	return committeeMock
}
