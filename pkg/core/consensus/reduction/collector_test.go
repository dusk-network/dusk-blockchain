package reduction

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.dusk.network/dusk-core/dusk-go/mocks"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"golang.org/x/crypto/ed25519"
)

// func TestReduction(t *testing.T) {
// 	eventBus := wire.New()
// 	committeeMock := mockCommittee(2, true, nil)
// 	timeOut := 100 * time.Millisecond

// 	broker := LaunchBlockReducer(eventBus, committeeMock, timeOut)
// 	// listen for outgoing votes of either kind, so we can verify they are being
// 	// sent out properly.
// 	outgoingReduction := make(chan *bytes.Buffer, 2)
// 	outgoingAgreement := make(chan *bytes.Buffer, 1)
// 	eventBus.Subscribe(msg.OutgoingBlockReductionTopic, outgoingReduction)
// 	eventBus.Subscribe(msg.OutgoingBlockAgreementTopic, outgoingAgreement)

// 	// update round
// 	updateRound(eventBus, 1)

// 	// send a hash to start reduction
// 	hash, _ := crypto.RandEntropy(32)
// 	broker.selectionChan <- hash

// 	// send mocked events until we get a result from the outgoingAgreement channel
// 	timer := time.After(2 * time.Second)
// 	for {
// 		select {
// 		case <-outgoingAgreement:
// 			// should have 2 reduction votes in the outgoingReduction channel
// 			assert.Equal(t, 2, len(outgoingReduction))
// 			// test successful
// 			return
// 		case <-timer:
// 			t.Fatal("reduction did not finish in time")
// 		default:
// 			ev := mockBlockEventBuffer(broker.ctx.state.Round(), broker.ctx.state.Step(),
// 				hash)
// 			eventBus.Publish(string(topics.BlockReduction), ev)

// 		}
// 	}
// }

func TestReductionTimeout(t *testing.T) {
	eventBus := wire.New()
	committeeMock := mockCommittee(2, true, nil)
	timeOut := 100 * time.Millisecond

	broker := LaunchBlockReducer(eventBus, committeeMock, timeOut)

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
	broker.selectionChan <- hash

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

func mockBlockEventBuffer(round uint64, step uint8, hash []byte) *bytes.Buffer {
	keys, _ := user.NewRandKeys()
	signedHash, _ := bls.Sign(keys.BLSSecretKey, keys.BLSPubKey, hash)
	marshaller := committee.NewReductionEventUnMarshaller(nil)

	bev := &committee.ReductionEvent{
		EventHeader: &consensus.EventHeader{
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

func mockCommittee(quorum int, isMember bool, verification error) committee.Committee {
	committeeMock := &mocks.Committee{}
	committeeMock.On("Quorum").Return(quorum)
	committeeMock.On("IsMember", mock.AnythingOfType("[]uint8")).Return(isMember)
	return committeeMock
}

func updateRound(eventBus *wire.EventBus, round uint64) {
	roundBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(roundBytes[:8], round)
	eventBus.Publish(msg.RoundUpdateTopic, bytes.NewBuffer(roundBytes))
}
