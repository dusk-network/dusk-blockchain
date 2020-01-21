package peer

import (
	"bytes"
	"fmt"

	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/dupemap"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing/chainsync"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/responding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	log "github.com/sirupsen/logrus"
)

type lightRouter struct {
	publisher eventbus.Publisher
	dupeMap   *dupemap.DupeMap

	dataRequestor *responding.DataRequestor
	synchronizer  *chainsync.ChainSynchronizer
	ponger        processing.Ponger

	peerInfo string
}

func newLightRouter(publisher eventbus.Publisher, dupeMap *dupemap.DupeMap, db database.DB, rpcBus *rpcbus.RPCBus, counter *chainsync.Counter, responseChan chan<- *bytes.Buffer, peerInfo string) *lightRouter {
	return &lightRouter{
		publisher:     publisher,
		dupeMap:       dupeMap,
		dataRequestor: responding.NewDataRequestor(db, rpcBus, responseChan),
		synchronizer:  chainsync.NewChainSynchronizer(publisher, rpcBus, responseChan, counter),
		ponger:        processing.NewPonger(responseChan),
	}
}

func (l *lightRouter) Collect(b *bytes.Buffer) error {
	topic, err := topics.Extract(b)
	if err != nil {
		return err
	}
	l.route(topic, b)
	return nil
}

func (l *lightRouter) route(topic topics.Topic, b *bytes.Buffer) {
	var err error
	switch topic {
	case topics.Inv:
		err = l.dataRequestor.RequestMissingItems(b)
	case topics.Ping:
		l.ponger.Pong()
	case topics.Block:
		err = l.synchronizer.Synchronize(b, l.peerInfo)
	default:
		err = fmt.Errorf("topic unroutable: %s", topic.String())
	}

	if err != nil {
		log.WithFields(log.Fields{
			"process": "peer",
			"error":   err,
		}).Errorf("problem handling message %s", topic.String())
	}
}
