package peer

import (
	"bytes"
	"fmt"

	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing/chainsync"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/responding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	log "github.com/sirupsen/logrus"
)

type lightRouter struct {
	publisher eventbus.Publisher

	dataRequestor *responding.DataRequestor
	synchronizer  *chainsync.ChainSynchronizer
	ponger        processing.Ponger

	peerInfo string
}

func newLightRouter(publisher eventbus.Publisher, db database.DB, rpcBus *rpcbus.RPCBus, counter *chainsync.Counter, responseChan chan<- *bytes.Buffer, peerInfo string) *lightRouter {
	return &lightRouter{
		publisher:     publisher,
		dataRequestor: responding.NewDataRequestor(db, rpcBus, responseChan),
		synchronizer:  chainsync.NewChainSynchronizer(publisher, rpcBus, responseChan, counter),
		ponger:        processing.NewPonger(responseChan),
	}
}

func (l *lightRouter) Collect(packet []byte) error {
	b := bytes.NewBuffer(packet)
	msg, err := message.Unmarshal(b)
	if err != nil {
		return err
	}
	l.route(*b, msg)
	return nil
}

func (l *lightRouter) route(b bytes.Buffer, msg message.Message) {
	var err error
	category := msg.Category()
	switch category {
	case topics.Inv:
		err = l.dataRequestor.RequestMissingItems(&b)
	case topics.Ping:
		l.ponger.Pong()
	case topics.Block:
		err = l.synchronizer.Synchronize(&b, l.peerInfo)
	case topics.Pong:
		// Just here to avoid the error. We don't do anything with Pong
	default:
		err = fmt.Errorf("topic unroutable: %s", category.String())
	}

	if err != nil {
		log.WithFields(log.Fields{
			"process": "peer",
			"error":   err,
		}).Errorf("problem handling message %s", category.String())
	}
}
