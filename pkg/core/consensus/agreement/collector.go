package agreement

import (
	"bytes"
	"encoding/binary"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/msg"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
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

func getInitialRound(eventBus wire.EventBroker) uint64 {
	initChannel := make(chan uint64, 1)
	initCollector := &initCollector{initChannel}
	go wire.NewTopicListener(eventBus, initCollector, msg.InitializationTopic).Accept()

	// Wait for the initial round to be published
	round := <-initChannel
	log.WithFields(log.Fields{
		"process": "factory",
		"round":   round,
	}).Debug("Received initial round")
	return round
}

func initReductionResultCollector(subscriber wire.EventSubscriber) <-chan voteSet {
	resultChan := make(chan voteSet, 1)
	collector := &reductionResultCollector{resultChan, reduction.NewUnMarshaller()}
	go wire.NewTopicListener(subscriber, collector, msg.ReductionResultTopic).Accept()
	return resultChan
}

func (r *reductionResultCollector) Collect(m *bytes.Buffer) error {
	var round uint64
	if err := encoding.ReadUint64(m, binary.LittleEndian, &round); err != nil {
		return err
	}

	votes, err := r.reductionUnmarshaller.UnmarshalVoteSet(m)
	if err != nil {
		return err
	}

	r.resultChan <- voteSet{round, votes}
	return nil
}
