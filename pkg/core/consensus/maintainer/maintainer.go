package maintainer

import (
	"bytes"
	"fmt"

	"github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/wallet"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	log "github.com/sirupsen/logrus"
)

var l = log.WithField("process", "StakeAutomaton")

// The StakeAutomaton is a process that keeps note of when certain consensus transactions
// expire, and makes sure the node remains within the bidlist/committee, when those
// transactions are close to expiring.
type StakeAutomaton struct {
	eventBroker eventbus.Broker
	rpcBus      *rpcbus.RPCBus
	roundChan   <-chan consensus.RoundUpdate

	pubKeyBLS []byte
	m         ristretto.Scalar
	p         user.Provisioners
	bidList   user.BidList

	bidEndHeight   uint64
	stakeEndHeight uint64
}

// How many blocks away from expiration the transactions should be
// renewed.
const renewalOffset = 100

// New creates a new instance of StakeAutomaton that is used to automate the
// resending of stakes and alleviate the burden for a user to having to
// manually manage restaking
func New(eventBroker eventbus.Broker, rpcBus *rpcbus.RPCBus, pubKeyBLS []byte, m ristretto.Scalar) *StakeAutomaton {
	return &StakeAutomaton{
		eventBroker:    eventBroker,
		rpcBus:         rpcBus,
		roundChan:      consensus.InitRoundUpdate(eventBroker),
		pubKeyBLS:      pubKeyBLS,
		m:              m,
		bidEndHeight:   1,
		stakeEndHeight: 1,
	}
}

// Listen to round updates and takes the proper decision Stake-wise
func (m *StakeAutomaton) Listen() {
	for roundUpdate := range m.roundChan {
		// Rehydrate consensus state
		m.p = roundUpdate.P
		m.bidList = roundUpdate.BidList

		// TODO: handle new provisioners and bidlist coming from roundupdate
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

func (m *StakeAutomaton) findMostRecentBid() uint64 {
	var highest uint64
	for _, bid := range m.bidList {
		if bytes.Equal(m.m.Bytes(), bid.M[:]) && bid.EndHeight > highest {
			highest = bid.EndHeight
		}
	}

	return highest
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

func (m *StakeAutomaton) sendBid() error {
	amount, lockTime := m.getTxSettings()
	if amount == 0 || lockTime == 0 {
		return fmt.Errorf("invalid settings: amount: %v / locktime: %v", amount, lockTime)
	}

	l.WithFields(log.Fields{
		"amount":   amount,
		"locktime": lockTime,
	}).Tracef("Sending bid tx")

	//TODO #363
	//req := &node.ConsensusTxRequest{Amount: amount, LockTime: lockTime}
	//_, err := m.rpcBus.Call(topics.SendBidTx, rpcbus.NewRequest(req), 0)
	//if err != nil {
	//	return err
	//}

	return nil
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

	var err error
	//TODO #363
	//req := &node.ConsensusTxRequest{Amount: amount, LockTime: lockTime}
	//_, err := m.rpcBus.Call(topics.SendStakeTx, rpcbus.NewRequest(req), 0)
	return err
}

func (m *StakeAutomaton) getTxSettings() (uint64, uint64) {
	settings := config.Get().Consensus
	amount := settings.DefaultAmount
	lockTime := settings.DefaultLockTime
	if lockTime > transactions.MaxLockTime {
		l.Warnf("default locktime was configured to be greater than the maximum (%v) - defaulting to %v", lockTime, transactions.MaxLockTime)
		lockTime = transactions.MaxLockTime
	}

	// Convert amount from atomic units to whole units of DUSK
	amount = amount * wallet.DUSK

	return amount, lockTime
}
