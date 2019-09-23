package agreement

import (
	"bytes"
	"encoding/binary"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/msg"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	log "github.com/sirupsen/logrus"
)

type initCollector struct {
	initChannel chan uint64
}

type reductionResultCollector struct {
	resultChan            chan voteSet
	reductionUnmarshaller *reduction.UnMarshaller
}

type voteSet struct {
	round uint64
	votes []wire.Event
}

func (i *initCollector) Collect(roundBuffer *bytes.Buffer) error {
	round := binary.LittleEndian.Uint64(roundBuffer.Bytes())
	i.initChannel <- round
	return nil
}

func getInitialRound(eventBus eventbus.Broker) uint64 {
	initChannel := make(chan uint64, 1)
	initCollector := &initCollector{initChannel}
	go eventbus.NewTopicListener(eventBus, initCollector, msg.InitializationTopic).Accept()

	// Wait for the initial round to be published
	round := <-initChannel
	log.WithFields(log.Fields{
		"process": "factory",
		"round":   round,
	}).Debug("Received initial round")
	return round
}

func initReductionResultCollector(subscriber eventbus.Subscriber) <-chan voteSet {
	resultChan := make(chan voteSet, 1)
	collector := &reductionResultCollector{resultChan, reduction.NewUnMarshaller()}
	go eventbus.NewTopicListener(subscriber, collector, msg.ReductionResultTopic).Accept()
	return resultChan
}

func (r *reductionResultCollector) Collect(m *bytes.Buffer) error {
	var round uint64
	if err := encoding.ReadUint64(m, binary.LittleEndian, &round); err != nil {
		return err
	}

	// We drop the error here. If there is no vote set included in the message,
	// we still need to trigger `sendAgreement` for the step counter to stay
	// in sync.
	votes, _ := r.reductionUnmarshaller.UnmarshalVoteSet(m)

	r.resultChan <- voteSet{round, votes}
	return nil
}
