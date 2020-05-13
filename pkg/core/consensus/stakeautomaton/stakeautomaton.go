package stakeautomaton

import (
	"context"
	"fmt"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/wallet"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-protobuf/autogen/go/node"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var l = log.WithField("process", "StakeAutomaton")

// StakeAutomaton is a process that keeps note of when certain consensus transactions
// expire, and makes sure the node remains within the bidlist/committee, when those
// transactions are close to expiring.
type StakeAutomaton struct {
	eventBroker eventbus.Broker
	rpcBus      *rpcbus.RPCBus
	roundChan   <-chan consensus.RoundUpdate

	pubKeyBLS []byte
	p         user.Provisioners
	height    uint64

	stakeEndHeight uint64

	running bool
}

// How many blocks away from expiration the transactions should be
// renewed.
const renewalOffset = 100

// New creates a new instance of StakeAutomaton that is used to automate the
// resending of stakes and alleviate the burden for a user to having to
// manually manage restaking
func New(eventBroker eventbus.Broker, rpcBus *rpcbus.RPCBus, pubKeyBLS []byte, srv *grpc.Server) *StakeAutomaton {
	a := &StakeAutomaton{
		eventBroker:    eventBroker,
		rpcBus:         rpcBus,
		pubKeyBLS:      pubKeyBLS,
		stakeEndHeight: 1,
		running:        false,
	}

	if srv != nil {
		node.RegisterMaintainerServer(srv, a)
	}
	return a
}

// AutomateConsensusTxs will automate the sending of stakes and bids.
func (m *StakeAutomaton) AutomateConsensusTxs(ctx context.Context, e *node.EmptyRequest) (*node.GenericResponse, error) {
	if !m.running {
		// We only initialize the `roundChan` here so that we don't clog the channel with
		// blocks while the maintainer is not actually running yet.
		m.roundChan = consensus.InitRoundUpdate(m.eventBroker)
		m.running = true
		go m.Listen()
	}

	return &node.GenericResponse{Response: "stake transactions are now being automated"}, nil
}

// Listen to round updates and takes the proper decision Stake-wise
func (m *StakeAutomaton) Listen() {
	for roundUpdate := range m.roundChan {
		// Rehydrate consensus state
		m.p = roundUpdate.P
		m.height = roundUpdate.Round

		if roundUpdate.Round+renewalOffset >= m.stakeEndHeight {
			endHeight := m.findMostRecentStake()

			// Only send stake if this is the first time we notice it's about to expire
			if endHeight > m.stakeEndHeight {
				m.stakeEndHeight = endHeight
			} else if m.stakeEndHeight != 0 {
				if err := m.sendStake(); err != nil {
					l.WithError(err).Warnln("could not send stake tx")
					continue
				}
				m.stakeEndHeight = 0
			}
		}
	}
}

func (m *StakeAutomaton) findMostRecentStake() uint64 {
	member := m.p.GetMember(m.pubKeyBLS)
	if member != nil {
		var highest uint64
		for _, stake := range member.Stakes {
			if stake.EndHeight > highest {
				highest = stake.EndHeight
			}
		}
		return highest
	}

	return 0
}

func (m *StakeAutomaton) sendStake() error {
	amount, lockTime := m.getTxSettings()
	if amount == 0 || lockTime == 0 {
		return fmt.Errorf("invalid settings: amount: %v / locktime: %v", amount, lockTime)
	}

	l.WithFields(log.Fields{
		"amount":   amount,
		"locktime": lockTime,
	}).Tracef("Sending stake tx")

	req := &node.StakeRequest{
		Amount:   amount,
		Fee:      config.MinFee,
		Locktime: lockTime,
	}
	_, err := m.rpcBus.Call(topics.SendStakeTx, rpcbus.NewRequest(req), 5*time.Second)
	return err
}

func (m *StakeAutomaton) getTxSettings() (uint64, uint64) {
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
