package agreement

import (
	"bytes"
	"encoding/binary"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	log "github.com/sirupsen/logrus"
)

type reductionResultCollector struct {
	resultChan            chan voteSet
	reductionUnmarshaller *reduction.UnMarshaller
}

type voteSet struct {
	round uint64
	votes []wire.Event
}

func getInitialRound(eventBus eventbus.Broker) uint64 {
	initChannel := make(chan bytes.Buffer, 1)
	listener := eventbus.NewChanListener(initChannel)
	id := eventBus.Subscribe(topics.Initialization, listener)

	roundBuf := <-initChannel
	round := binary.LittleEndian.Uint64(roundBuf.Bytes())
	log.WithFields(log.Fields{
		"process": "factory",
		"round":   round,
	}).Debug("Received initial round")

	// unsubscribe the listener as initialization messages are no longer relevant
	eventBus.Unsubscribe(topics.Initialization, id)
	return round
}

func initReductionResultCollector(subscriber eventbus.Subscriber) <-chan voteSet {
	resultChan := make(chan voteSet, 1)
	collector := &reductionResultCollector{resultChan, reduction.NewUnMarshaller()}
	eventbus.NewTopicListener(subscriber, collector, topics.ReductionResult, eventbus.ChannelType)
	return resultChan
}

func (r *reductionResultCollector) Collect(m bytes.Buffer) error {
	var round uint64
	if err := encoding.ReadUint64LE(&m, &round); err != nil {
		return err
	}

	// We drop the error here. If there is no vote set included in the message,
	// we still need to trigger `sendAgreement` for the step counter to stay
	// in sync.
	votes, _ := r.reductionUnmarshaller.UnmarshalVoteSet(&m)

	r.resultChan <- voteSet{round, votes}
	return nil
}
