package peer

import (
	"bytes"
	"fmt"

	"github.com/dusk-network/dusk-blockchain/pkg/util/diagnostics"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/dupemap"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/processing/chainsync"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/responding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
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
		err = m.dataBroker.SendTxsItems()
	case topics.Inv:
		err = m.dataRequestor.RequestMissingItems(&b)
	case topics.Block:
		err = m.synchronizer.HandleBlock(&b, m.peerInfo)
	case topics.Ping:
		m.ponger.Pong()
	case topics.Pong:
		// Just here to avoid the error message, as pong is unroutable but
		// otherwise carries no relevant information beyond the receiving
		// of this message
	case topics.GetRoundResults:
		err = m.roundResultBroker.ProvideRoundResult(&b)
	case topics.GetCandidate:
		// We only accept a certain request once, to avoid infinitely
		// requesting the same block
		// TODO: interface - buffer should be immutable. Change the dupemap to
		// deal with values rather than reference
		if m.dupeMap.CanFwd(bytes.NewBuffer(msg.Id())) {
			err = m.candidateBroker.ProvideCandidate(&b)
		}
	case topics.Candidate:
		// We don't use the dupe map to prevent infinite dissemination,
		// as it could deprive of us receiving a candidate we might
		// need later, but which was discarded initially.
		// The candidate component will use it's own repropagation rules.
		errList := m.publisher.Publish(category, msg)
		diagnostics.LogPublishErrors("peer/router.go, topics.Candidate", errList)

	default:
		if m.CanRoute(category) {
			if m.dupeMap.CanFwd(bytes.NewBuffer(msg.Id())) {
				errList := m.publisher.Publish(category, msg)
				diagnostics.LogPublishErrors("peer/router.go, default", errList)

			}
		} else {
			err = fmt.Errorf("%s topic not routable", category.String())
		}
	}

	return err
}
