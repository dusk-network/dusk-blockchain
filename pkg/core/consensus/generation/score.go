package generation

import (
	"bytes"
	"encoding/hex"

	"gitlab.dusk.network/dusk-core/zkproof"

	"github.com/bwesterb/go-ristretto"
	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/selection"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

// LaunchScoreGenerationComponent will start the processes for score generation.
func LaunchScoreGenerationComponent(eventBus *wire.EventBus, d, k ristretto.Scalar) *broker {
	broker := newBroker(eventBus, d, k)
	go broker.Listen()
	return broker
}

type (
	forwarder struct {
		publisher  wire.EventPublisher
		marshaller wire.EventMarshaller
		state      consensus.State
		seeds      map[string]uint64
	}

	broker struct {
		proofGenerator *proofGenerator
		forwarder      *forwarder

		// subscriber channels
		roundChannel   <-chan uint64
		bidListChannel <-chan user.BidList
	}
)

func newForwarder(publisher wire.EventPublisher) *forwarder {
	return &forwarder{
		publisher:  publisher,
		marshaller: &selection.ScoreUnMarshaller{},
		state:      consensus.NewState(),
		seeds:      make(map[string]uint64),
	}
}

func (f *forwarder) forward(proof zkproof.ZkProof, seed []byte) {
	if bytes.Equal(seed, f.getLatestSeed()) {
		f.generateScoreEvent(proof)
	}
}

func (f *forwarder) getLatestSeed() []byte {
	for seed, round := range f.seeds {
		if round == f.state.Round() {
			seedBytes, err := hex.DecodeString(seed)
			if err != nil {
				panic(err)
			}

			return seedBytes
		}
	}

	panic("uninitialized forwarder")
}

func (f *forwarder) generateScoreEvent(proof zkproof.ZkProof) {
	// TODO: get an actual hash by generating a block
	hash, err := crypto.RandEntropy(32)
	if err != nil {
		panic(err)
	}

	sev := selection.ScoreEvent{
		Round:         f.state.Round(),
		Score:         proof.Score,
		Proof:         proof.Proof,
		Z:             proof.Z,
		BidListSubset: proof.BinaryBidList,
		Seed:          f.getLatestSeed(),
		VoteHash:      hash,
	}

	marshalledEvent := f.marshalScore(sev)
	log.WithFields(log.Fields{
		"process":         "generation",
		"collector round": f.state.Round(),
	}).Debugln("sending proof")
	f.publisher.Publish(string(topics.Gossip), marshalledEvent)
}

func (f *forwarder) marshalScore(sev selection.ScoreEvent) *bytes.Buffer {
	buffer := new(bytes.Buffer)
	if err := f.marshaller.Marshal(buffer, sev); err != nil {
		panic(err)
	}

	message, err := wire.AddTopic(buffer, topics.Score)
	if err != nil {
		panic(err)
	}

	return message
}

func (f *forwarder) generateSeed(round uint64) []byte {
	f.state.Update(round)
	// TODO: make an actual seed by signing the previous block seed
	seed, _ := crypto.RandEntropy(33)
	f.seeds[hex.EncodeToString(seed)] = round
	return seed
}

func newBroker(eventBroker wire.EventBroker, d, k ristretto.Scalar) *broker {
	proofGenerator := newProofGenerator(d, k)
	roundChannel := consensus.InitRoundUpdate(eventBroker)
	bidListChannel := selection.InitBidListUpdate(eventBroker)
	broker := &broker{
		proofGenerator: proofGenerator,
		roundChannel:   roundChannel,
		bidListChannel: bidListChannel,
		forwarder:      newForwarder(eventBroker),
	}

	go wire.NewTopicListener(eventBroker, broker, msg.BlockGenerationTopic).Accept()
	return broker
}

func (b *broker) Listen() {
	for {
		select {
		case round := <-b.roundChannel:
			seed := b.forwarder.generateSeed(round)
			proof := b.proofGenerator.generateProof(seed)
			b.forwarder.forward(proof, seed)
		case bidList := <-b.bidListChannel:
			b.proofGenerator.updateBidList(bidList)
		}
	}
}

func (b *broker) Collect(m *bytes.Buffer) error {
	seed := b.forwarder.getLatestSeed()
	proof := b.proofGenerator.generateProof(seed)
	b.forwarder.forward(proof, seed)
	return nil
}
