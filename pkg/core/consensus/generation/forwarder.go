package generation

import (
	"bytes"

	log "github.com/sirupsen/logrus"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/selection"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
	"gitlab.dusk.network/dusk-core/zkproof"
)

type forwarder struct {
	publisher      wire.EventPublisher
	blockGenerator *blockGenerator
	threshold      *consensus.Threshold
}

func newForwarder(publisher wire.EventPublisher, blockGenerator *blockGenerator) *forwarder {
	return &forwarder{
		publisher:      publisher,
		blockGenerator: blockGenerator,
		threshold:      consensus.NewThreshold(),
	}
}

func (f *forwarder) forwardScoreEvent(proof zkproof.ZkProof, round uint64, seed []byte) {
	// if our score is too low, don't bother
	if !f.threshold.Exceeds(proof.Score) {
		return
	}

	blk := f.blockGenerator.generateBlock(round, seed)
	if err := blk.SetHash(); err != nil {

		panic(err)
	}

	sev := &selection.ScoreEvent{
		Round:         round,
		Score:         proof.Score,
		Proof:         proof.Proof,
		Z:             proof.Z,
		BidListSubset: proof.BinaryBidList,
		Seed:          seed,
		VoteHash:      blk.Header.Hash,
	}

	marshalledEvent := f.marshalScore(sev)
	log.WithFields(log.Fields{
		"process":         "generation",
		"collector round": round,
	}).Debugln("sending proof")
	f.publisher.Stream(string(topics.Gossip), marshalledEvent)
	f.publisher.Stream(string(topics.Gossip), f.marshalBlock(blk))
}

func (f *forwarder) marshalScore(sev *selection.ScoreEvent) *bytes.Buffer {
	buffer := new(bytes.Buffer)
	if err := selection.MarshalScoreEvent(buffer, sev); err != nil {
		panic(err)
	}

	copy := *buffer
	f.publisher.Publish(string(topics.Score), &copy)
	message, err := wire.AddTopic(buffer, topics.Score)
	if err != nil {
		panic(err)
	}

	return message
}

func (f *forwarder) marshalBlock(blk *block.Block) *bytes.Buffer {
	buffer := new(bytes.Buffer)
	if err := blk.Encode(buffer); err != nil {
		panic(err)
	}

	copy := *buffer
	f.publisher.Publish(string(topics.Candidate), &copy)
	message, err := wire.AddTopic(buffer, topics.Candidate)
	if err != nil {
		panic(err)
	}

	return message
}
