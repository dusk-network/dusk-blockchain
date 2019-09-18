package maintainer

import (
	"bytes"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/transactor"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/wallet/transactions"
	log "github.com/sirupsen/logrus"
)

var l = log.WithField("process", "maintainer")

// The maintainer is a process that keeps note of when certain consensus transactions
// expire, and makes sure the node remains within the bidlist/committee, when those
// transactions are close to expiring.
type maintainer struct {
	eventBroker wire.EventBroker
	roundChan   <-chan consensus.RoundUpdate

	pubKeyBLS []byte
	m         ristretto.Scalar
	p         user.Stakers
	bidList   user.BidList

	bidEndHeight   uint64
	stakeEndHeight uint64

	amount, lockTime, offset uint64

	transactor *transactor.Transactor
}

func newMaintainer(eventBroker wire.EventBroker, pubKeyBLS []byte, m ristretto.Scalar, transactor *transactor.Transactor, amount, lockTime, offset uint64) (*maintainer, error) {
	return &maintainer{
		eventBroker: eventBroker,
		roundChan:   consensus.InitRoundUpdate(eventBroker),
		pubKeyBLS:   pubKeyBLS,
		m:           m,
		transactor:  transactor,
		amount:      amount,
		lockTime:    lockTime,
		offset:      offset,
	}, nil
}

func Launch(eventBroker wire.EventBroker, pubKeyBLS []byte, m ristretto.Scalar, transactor *transactor.Transactor, amount, lockTime, offset uint64) error {

	maintainer, err := newMaintainer(eventBroker, pubKeyBLS, m, transactor, amount, lockTime, offset)
	if err != nil {
		return err
	}

	// Let's see if we have active stakes and bids already
	maintainer.bidEndHeight = maintainer.findMostRecentBid()
	// If not, we create them here
	if maintainer.bidEndHeight == 0 {
		if err := maintainer.sendBid(); err != nil {
			l.WithError(err).Warnln("could not send bid tx")
			// Set end height to 1 so we retry sending the tx in `listen`
			maintainer.bidEndHeight = 1
		}
	}

	maintainer.stakeEndHeight = maintainer.findMostRecentStake()
	if maintainer.stakeEndHeight == 0 {
		if err := maintainer.sendStake(); err != nil {
			l.WithError(err).Warnln("could not send stake tx")
			maintainer.stakeEndHeight = 1
		}
	}

	// Set up listening loop
	go maintainer.listen()
	return nil
}

func (m *maintainer) listen() {
	for {
		select {
		case roundUpdate := <-m.roundChan:
			// Rehydrate consensus state
			m.p = roundUpdate.P
			m.bidList = roundUpdate.BidList

			// TODO: handle new provisioners and bidlist coming from roundupdate
			if roundUpdate.Round+m.offset >= m.bidEndHeight {
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

			if roundUpdate.Round+m.offset >= m.stakeEndHeight {
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
}

func (m *maintainer) findMostRecentBid() uint64 {
	var highest uint64
	for _, bid := range m.bidList {
		if bytes.Equal(m.m.Bytes(), bid.M[:]) && bid.EndHeight > highest {
			highest = bid.EndHeight
		}
	}

	return highest
}

func (m *maintainer) findMostRecentStake() uint64 {
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

func (m *maintainer) sendBid() error {
	bid, err := m.transactor.CreateBidTx(m.amount, m.lockTime)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	if err := transactions.Marshal(buf, bid); err != nil {
		return err
	}

	m.eventBroker.Publish(string(topics.Tx), buf)
	return nil
}

func (m *maintainer) sendStake() error {
	stake, err := m.transactor.CreateStakeTx(m.amount, m.lockTime)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	if err := transactions.Marshal(buf, stake); err != nil {
		return err
	}

	m.eventBroker.Publish(string(topics.Tx), buf)
	return nil
}
