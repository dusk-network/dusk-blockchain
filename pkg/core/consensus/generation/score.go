package generation

import (
	"bytes"

	"github.com/bwesterb/go-ristretto"
	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/selection"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
	"gitlab.dusk.network/dusk-core/zkproof"
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
	}

	broker struct {
		proofGenerator *proofGenerator
		forwarder      *forwarder
		seeder         *seeder

		// subscriber channels
		roundChannel   <-chan uint64
		bidListChannel <-chan user.BidList
	}
)

func newForwarder(publisher wire.EventPublisher) *forwarder {
	return &forwarder{
		publisher:  publisher,
		marshaller: &selection.ScoreUnMarshaller{},
	}
}

func (f *forwarder) forwardScoreEvent(proof zkproof.ZkProof, round uint64, seed []byte) {
	// TODO: get an actual hash by generating a block
	hash, err := crypto.RandEntropy(32)
	if err != nil {
		panic(err)
	}

	sev := selection.ScoreEvent{
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

func newBroker(eventBroker wire.EventBroker, d, k ristretto.Scalar) *broker {
	proofGenerator := newProofGenerator(d, k)
	roundChannel := consensus.InitRoundUpdate(eventBroker)
	bidListChannel := selection.InitBidListUpdate(eventBroker)
	broker := &broker{
		proofGenerator: proofGenerator,
		roundChannel:   roundChannel,
		bidListChannel: bidListChannel,
		forwarder:      newForwarder(eventBroker),
		seeder:         &seeder{},
	}

	go wire.NewTopicListener(eventBroker, broker, msg.BlockGenerationTopic).Accept()
	return broker
}

func (b *broker) Listen() {
	for {
		select {
		case round := <-b.roundChannel:
			seed := b.seeder.GenerateSeed(round)
			proof := b.proofGenerator.generateProof(seed)
			b.Forward(proof, seed)
		case bidList := <-b.bidListChannel:
			b.proofGenerator.updateBidList(bidList)
		}
	}
}

func (b *broker) Collect(m *bytes.Buffer) error {
	seed := b.seeder.LatestSeed()
	proof := b.proofGenerator.generateProof(seed)
	b.Forward(proof, seed)
	return nil
}

func (b *broker) Forward(proof zkproof.ZkProof, seed []byte) {
	b.seeder.RLock()
	if b.seeder.isFresh(seed) {
		round := b.seeder.round
		b.seeder.RUnlock()
		b.forwarder.forwardScoreEvent(proof, round, seed)
		return
	}

	b.seeder.RUnlock()
}
