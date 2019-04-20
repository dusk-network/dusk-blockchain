package generation

import (
	"bytes"

	log "github.com/sirupsen/logrus"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/selection"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
	"gitlab.dusk.network/dusk-core/zkproof"
)

type forwarder struct {
	publisher wire.EventPublisher
}

func newForwarder(publisher wire.EventPublisher) *forwarder {
	return &forwarder{
		publisher: publisher,
	}
}

func (f *forwarder) forwardScoreEvent(proof zkproof.ZkProof, round uint64, seed []byte) {
	// TODO: get an actual hash by generating a block
	hash, err := crypto.RandEntropy(32)
	if err != nil {
		panic(err)
	}

	sev := &selection.ScoreEvent{
		Round:         round,
		Score:         proof.Score,
		Proof:         proof.Proof,
		Z:             proof.Z,
		BidListSubset: proof.BinaryBidList,
		Seed:          seed,
		VoteHash:      hash,
	}

	marshalledEvent := f.marshalScore(sev)
	log.WithFields(log.Fields{
		"process":         "generation",
		"collector round": round,
	}).Debugln("sending proof")
	f.publisher.Publish(string(topics.Gossip), marshalledEvent)
}

func (f *forwarder) marshalScore(sev *selection.ScoreEvent) *bytes.Buffer {
	buffer := new(bytes.Buffer)
	if err := selection.MarshalScoreEvent(buffer, sev); err != nil {
		panic(err)
	}

	message, err := wire.AddTopic(buffer, topics.Score)
	if err != nil {
		panic(err)
	}

	return message
}
