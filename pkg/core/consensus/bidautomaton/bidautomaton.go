package bidautomaton

import (
	"context"
	"fmt"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/wallet"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-protobuf/autogen/go/node"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var l = log.WithField("process", "BidAutomaton")

type BidAutomaton struct {
	eventBroker eventbus.Broker
	client      node.TransactorClient
	roundChan   <-chan consensus.RoundUpdate

	edPK   []byte
	height uint64

	bidEndHeight uint64

	running bool
}

// How many blocks away from expiration the transactions should be
// renewed.
const renewalOffset = 100

func New(eventBroker eventbus.Broker, client node.TransactorClient, edPK []byte, srv *grpc.Server) *BidAutomaton {
	a := &BidAutomaton{
		eventBroker:  eventBroker,
		client:       client,
		edPK:         edPK,
		bidEndHeight: 1,
		running:      false,
	}

	if srv != nil {
		node.RegisterMaintainerServer(srv, a)
	}
	return a
}

// TODO: update protobuf to have a separate method from stake automaton
func (m *BidAutomaton) AutomateConsensusTxs(ctx context.Context, e *node.EmptyRequest) (*node.GenericResponse, error) {
	if !m.running {
		m.roundChan = consensus.InitRoundUpdate(m.eventBroker)
		m.running = true
		go m.Listen()
	}

	return &node.GenericResponse{Response: "Bid transactions are now being automated"}, nil
}

func (m *BidAutomaton) Listen() {
	for roundUpdate := range m.roundChan {
		// Rehydrate consensus state
		m.height = roundUpdate.Round

		if roundUpdate.Round+renewalOffset >= m.bidEndHeight {
			endHeight := m.findMostRecentBid()

			// Only send bid if this is the first time we notice it's about to expire
			if endHeight > m.bidEndHeight {
				m.bidEndHeight = endHeight
			} else if m.bidEndHeight != 0 {
				if err := m.sendBid(); err != nil {
					l.WithError(err).Warnln("could not send bid tx")
					continue
				}
				// Set end height to 0 to ensure we only send a transaction once
				m.bidEndHeight = 0
			}
		}
	}
}

// nolint
func (m *BidAutomaton) sendBid() error {
	amount, lockTime := m.getTxSettings()
	if amount == 0 || lockTime == 0 {
		return fmt.Errorf("invalid settings: amount: %v / locktime: %v", amount, lockTime)
	}

	l.WithFields(log.Fields{
		"amount":   amount,
		"locktime": lockTime,
	}).Tracef("Sending bid tx")

	_, err := m.client.Bid(context.Background(), &node.BidRequest{Amount: amount, Fee: config.MinFee, Locktime: lockTime})
	return err
}

func (m *BidAutomaton) getTxSettings() (uint64, uint64) {
	settings := config.Get().Consensus
	amount := settings.DefaultAmount
	lockTime := settings.DefaultLockTime

	if lockTime > config.MaxLockTime {
		l.Warnf("default locktime exceeds maximum (%v) - defaulting to %v", lockTime, config.MaxLockTime)
		lockTime = config.MaxLockTime
	}

	// Convert amount from atomic units to whole units of DUSK
	amount = amount * wallet.DUSK

	return amount, lockTime
}
