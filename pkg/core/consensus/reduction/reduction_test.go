package reduction

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.dusk.network/dusk-core/dusk-go/mocks"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/events"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/selection"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
	"golang.org/x/crypto/ed25519"
)

// func init() {
// log.SetLevel(log.DebugLevel)
// }

func TestReduction(t *testing.T) {
	// send a hash to start reduction
	hash, _ := crypto.RandEntropy(32)
	eventBus := wire.NewEventBus()
	committeeMock := mockCommittee(2, true)
	timeOut := 200 * time.Millisecond

	broker := LaunchReducer(eventBus, committeeMock, timeOut)
	// listen for outgoing votes of either kind, so we can verify they are being
	// sent out properly.
	outgoingReduction := make(chan *bytes.Buffer, 2)
	outgoingAgreement := make(chan *bytes.Buffer, 1)
	eventBus.Subscribe(msg.OutgoingBlockReductionTopic, outgoingReduction)
	eventBus.Subscribe(msg.OutgoingBlockAgreementTopic, outgoingAgreement)

	// update round
	updateRound(eventBus, 1)

	// here we try to force the selection message to ALWAYS come after the round update
	for {
		if broker.ctx.state.Round() == 0 {
			time.Sleep(3 * time.Millisecond)
			continue
		}

		// now we can send the selection
		bestScoreBuf := mockSelectionEventBuffer(hash)
		eventBus.Publish(msg.BestScoreTopic, bestScoreBuf)
		break
	}

	// send mocked events until we get a result from the outgoingAgreement channel
	timer := time.After(2 * time.Second)
	count := 0
	for {
		select {
		case <-outgoingAgreement:
			// should have 2 reduction votes in the outgoingReduction channel
			assert.Equal(t, 2, len(outgoingReduction))
			// test successful
			return
		case <-timer:
			t.Fatal("reduction did not finish in time")
		default:
			count++
			// the committee is never more than 50 nodes. It does not make much sense to hammer the reducer more than that
			if count < 50 {
				ev := mockBlockEventBuffer(broker.ctx.state.Round(), broker.ctx.state.Step(), hash)
				eventBus.Publish(string(topics.Reduction), ev)
			}
		}
	}
}

func TestReductionTimeout(t *testing.T) {
	eventBus := wire.NewEventBus()
	committeeMock := mockCommittee(2, true)
	timeOut := 200 * time.Millisecond

	broker := LaunchReducer(eventBus, committeeMock, timeOut)

	// listen for outgoing votes of either kind, so we can verify they are being
	// sent out properly.
	outgoingReduction := make(chan *bytes.Buffer, 2)
	outgoingAgreement := make(chan *bytes.Buffer, 1)
	eventBus.Subscribe(msg.OutgoingBlockReductionTopic, outgoingReduction)
	eventBus.Subscribe(msg.OutgoingBlockAgreementTopic, outgoingAgreement)

	// update round
	updateRound(eventBus, 1)

	// send a hash to start reduction
	hash, _ := crypto.RandEntropy(32)

	// here we try to force the selection message to ALWAYS come after the round update
	for {
		if broker.ctx.state.Round() == 0 {
			time.Sleep(3 * time.Millisecond)
			continue
		}

		// now we can send the selection
		bestScoreBuf := mockSelectionEventBuffer(hash)
		eventBus.Publish(msg.BestScoreTopic, bestScoreBuf)
		break
	}

	timer := time.After(1 * time.Second)
	select {
	case <-outgoingAgreement:
		// this should not happen at all
		t.Fatal("got an agreement vote when quorum could have never been reached")
	case <-timer:
		// should have 2 reduction votes in the outgoingReduction channel
		// - one from the best score topic
		// - one from the first step timeout
		assert.Equal(t, 2, len(outgoingReduction))
	}
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
			PubKeyBLS: keys.BLSPubKey.Marshal(),
			Round:     round,
			Step:      step,
		},
		VotedHash:  hash,
		SignedHash: signedHash.Compress(),
	}

	buf := new(bytes.Buffer)
	_ = marshaller.Marshal(buf, bev)
	edSig := ed25519.Sign(*keys.EdSecretKey, buf.Bytes())
	completeBuf := bytes.NewBuffer(edSig)
	completeBuf.Write(keys.EdPubKeyBytes())
	completeBuf.Write(buf.Bytes())
	return completeBuf
}

func mockCommittee(quorum int, isMember bool) committee.Committee {
	committeeMock := &mocks.Committee{}
	committeeMock.On("Quorum").Return(quorum)
	committeeMock.On("IsMember",
		mock.AnythingOfType("[]uint8"),
		mock.AnythingOfType("uint64"),
		mock.AnythingOfType("uint8")).Return(isMember)
	return committeeMock
}

func updateRound(eventBus *wire.EventBus, round uint64) {
	roundBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(roundBytes[:8], round)
	eventBus.Publish(msg.RoundUpdateTopic, bytes.NewBuffer(roundBytes))
}
