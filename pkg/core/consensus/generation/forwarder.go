package generation

import (
	"bytes"
	"errors"

	log "github.com/sirupsen/logrus"

	"github.com/dusk-network/dusk-blockchain/pkg/core/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/selection"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	zkproof "github.com/dusk-network/dusk-zkproof"
)

type forwarder struct {
	publisher      eventbus.Publisher
	blockGenerator BlockGenerator
	threshold      *consensus.Threshold
	prevBlock      block.Block
}

func newForwarder(publisher eventbus.Publisher, blockGenerator BlockGenerator) *forwarder {
	return &forwarder{
		publisher:      publisher,
		blockGenerator: blockGenerator,
		threshold:      consensus.NewThreshold(),
	}
}

func (f *forwarder) setPrevBlock(blk block.Block) {
	f.prevBlock = blk
}

func (f *forwarder) forwardScoreEvent(proof zkproof.ZkProof, round uint64, seed []byte) error {
	// if our score is too low, don't bother
	if !f.threshold.Exceeds(proof.Score) {
		return errors.New("proof score too low")
	}

	blk, err := f.blockGenerator.GenerateBlock(round, seed, proof.Proof, proof.Score, f.prevBlock.Header.Hash)
	if err != nil {
		return err
	}

	// Retrieve and append the verified transactions from Mempool
	blockBytes := new(bytes.Buffer)
	if err = block.Marshal(blockBytes, blk); err != nil {
		return err
	}

	sev := &selection.ScoreEvent{
		Round:         round,
		Score:         proof.Score,
		Proof:         proof.Proof,
		Z:             proof.Z,
		BidListSubset: proof.BinaryBidList,
		PrevHash:      f.prevBlock.Header.Hash,
		Certificate:   f.prevBlock.Header.Certificate,
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
	return nil
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
	if err := block.Marshal(buffer, blk); err != nil {
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
