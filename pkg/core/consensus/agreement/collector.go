package agreement

import (
	"bytes"
	"encoding/binary"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/msg"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	log "github.com/sirupsen/logrus"
)

type initCollector struct {
	initChannel chan uint64
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
