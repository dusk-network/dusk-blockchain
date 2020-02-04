package peer

import (
	"bytes"
	"fmt"

	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/dupemap"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing/chainsync"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/responding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
)

// The messageRouter is connected to all of the processing units that are tied to the peer.
// It sends an incoming message in the right direction, according to its topic.
type messageRouter struct {
	publisher eventbus.Publisher
	dupeMap   *dupemap.DupeMap

	// 1-to-1 components
	blockHashBroker   *responding.BlockHashBroker
	dataRequestor     *responding.DataRequestor
	dataBroker        *responding.DataBroker
	roundResultBroker *responding.RoundResultBroker
	candidateBroker   *responding.CandidateBroker
	synchronizer      *chainsync.ChainSynchronizer
	ponger            processing.Ponger

	peerInfo string
	protocol.ServiceFlag
}

func newRouter(publisher eventbus.Publisher, dupeMap *dupemap.DupeMap, db database.DB, rpcBus *rpcbus.RPCBus, counter *chainsync.Counter, responseChan chan<- *bytes.Buffer, peerInfo string, serviceFlag protocol.ServiceFlag) *messageRouter {
	router := &messageRouter{
		publisher:       publisher,
		dupeMap:         dupeMap,
		blockHashBroker: responding.NewBlockHashBroker(db, responseChan),
		dataRequestor:   responding.NewDataRequestor(db, rpcBus, responseChan),
		dataBroker:      responding.NewDataBroker(db, rpcBus, responseChan),
		ponger:          processing.NewPonger(responseChan),
		peerInfo:        peerInfo,
		ServiceFlag:     serviceFlag,
	}

	// Components only needed for full node communication
	if serviceFlag == protocol.FullNode {
		router.roundResultBroker = responding.NewRoundResultBroker(rpcBus, responseChan)
		router.candidateBroker = responding.NewCandidateBroker(rpcBus, responseChan)
		router.synchronizer = chainsync.NewChainSynchronizer(publisher, rpcBus, responseChan, counter)
	}

	return router
}

func (m *messageRouter) Collect(packet []byte) error {
	b := bytes.NewBuffer(packet)
	msg, err := message.Unmarshal(b)
	if err != nil {
		return err
	}
	return m.route(*b, msg)
}

func (m *messageRouter) CanRoute(topic topics.Topic) bool {
	// Only accept Tx messages from light nodes
	if m.ServiceFlag == protocol.LightNode {
		return topic == topics.Tx
	}

	switch topic {
	case topics.Tx,
		topics.Candidate,
		topics.Score,
		topics.Reduction,
		topics.Agreement,
		topics.RoundResults:
		return true
	}

	return false
}

func (m *messageRouter) route(b bytes.Buffer, msg message.Message) error {
	var err error
	category := msg.Category()
	switch category {
	case topics.GetBlocks:
		err = m.blockHashBroker.AdvertiseMissingBlocks(&b)
	case topics.GetData:
		err = m.dataBroker.SendItems(&b)
	case topics.MemPool:
		if m.ServiceFlag == protocol.FullNode {
			err = m.dataBroker.SendTxsItems()
		}
	case topics.Inv:
		err = m.dataRequestor.RequestMissingItems(&b)
	case topics.Ping:
		m.ponger.Pong()
	case topics.Pong:
		// Just here to avoid the error message, as pong is unroutable but
		// otherwise carries no relevant information beyond the receiving
		// of this message

	// Full node topics
	case topics.Block:
		if m.ServiceFlag == protocol.FullNode {
			err = m.synchronizer.Synchronize(&b, m.peerInfo)
		}
	case topics.GetRoundResults:
		if m.ServiceFlag == protocol.FullNode {
			err = m.roundResultBroker.ProvideRoundResult(&b)
		}
	case topics.GetCandidate:
		if m.ServiceFlag == protocol.FullNode {
			// We only accept a certain request once, to avoid infinitely
			// requesting the same block
			// TODO: interface - buffer should be immutable. Change the dupemap to
			// deal with values rather than reference
			if m.dupeMap.CanFwd(bytes.NewBuffer(msg.Id())) {
				err = m.candidateBroker.ProvideCandidate(&b)
			}
		}
	case topics.Candidate:
		if m.ServiceFlag == protocol.FullNode {
			// We don't use the dupe map to prevent infinite dissemination,
			// as it could deprive of us receiving a candidate we might
			// need later, but which was discarded initially.
			// The candidate component will use it's own repropagation rules.
			m.publisher.Publish(category, msg)
		}
	default:
		if m.CanRoute(category) {
			if m.dupeMap.CanFwd(bytes.NewBuffer(msg.Id())) {
				m.publisher.Publish(category, msg)
			}
		} else {
			err = fmt.Errorf("%s topic not routable", category.String())
		}
	}

	return err
}
