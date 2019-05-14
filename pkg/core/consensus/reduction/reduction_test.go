package reduction

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

func TestReduction(t *testing.T) {
	// send a hash to start reduction
	hash, _ := crypto.RandEntropy(32)
	eventBus := wire.NewEventBus()
	committeeMock := mockCommittee(2, true, true)
	timeOut := 200 * time.Millisecond

	broker := LaunchReducer(eventBus, committeeMock, timeOut)
	// listen for outgoing votes of either kind, so we can verify they are being
	// sent out properly.
	outgoingReduction := make(chan *bytes.Buffer, 2)
	outgoingAgreement := make(chan *bytes.Buffer, 1)
	eventBus.Subscribe(msg.OutgoingBlockReductionTopic, outgoingReduction)
	eventBus.Subscribe(msg.OutgoingBlockAgreementTopic, outgoingAgreement)

	// update round
	consensus.UpdateRound(eventBus, 1)

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

func TestNoPublishingIfNotInCommittee(t *testing.T) {
	// send a hash to start reduction
	hash, _ := crypto.RandEntropy(32)
	eventBus := wire.NewEventBus()
	committeeMock := mockCommittee(2, true, false)
	timeOut := 200 * time.Millisecond

	broker := LaunchReducer(eventBus, committeeMock, timeOut)
	// listen for outgoing votes of either kind, so we can verify they are being
	// sent out properly.
	outgoingReduction := make(chan *bytes.Buffer, 2)
	outgoingAgreement := make(chan *bytes.Buffer, 1)
	eventBus.Subscribe(msg.OutgoingBlockReductionTopic, outgoingReduction)
	eventBus.Subscribe(msg.OutgoingBlockAgreementTopic, outgoingAgreement)

	// update round
	consensus.UpdateRound(eventBus, 1)

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
	timer := time.After(1 * time.Second)
	count := 0
	for {
		select {
		case <-outgoingAgreement:
			t.Fatal("message should not have been sent as AmMember is set to false")
			return
		case <-timer:
			//All good
			return
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
	committeeMock := mockCommittee(2, true, true)
	timeOut := 200 * time.Millisecond

	broker := LaunchReducer(eventBus, committeeMock, timeOut)

	// listen for outgoing votes of either kind, so we can verify they are being
	// sent out properly.
	outgoingReduction := make(chan *bytes.Buffer, 2)
	outgoingAgreement := make(chan *bytes.Buffer, 1)
	eventBus.Subscribe(msg.OutgoingBlockReductionTopic, outgoingReduction)
	eventBus.Subscribe(msg.OutgoingBlockAgreementTopic, outgoingAgreement)

	// update round
	consensus.UpdateRound(eventBus, 1)

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
