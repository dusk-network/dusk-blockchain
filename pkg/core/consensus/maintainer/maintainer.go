package maintainer

import (
	"bytes"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/committee"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
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
	roundChan   <-chan uint64
	bidChan     <-chan user.Bid

	pubKeyBLS []byte
	m         ristretto.Scalar
	c         *committee.Store
	bidList   *user.BidList

	bidEndHeight   uint64
	stakeEndHeight uint64

	amount, lockTime, offset uint64

	transactor *transactor.Transactor
}

func newMaintainer(eventBroker wire.EventBroker, db database.DB, pubKeyBLS []byte, m ristretto.Scalar, transactor *transactor.Transactor, amount, lockTime, offset uint64) (*maintainer, error) {
	if db == nil {
		_, db = heavy.CreateDBConnection()
	}

	bidList, err := user.NewBidList(db)
	if err != nil {
		return nil, err
	}

	return &maintainer{
		eventBroker: eventBroker,
		bidChan:     consensus.InitBidListUpdate(eventBroker),
		roundChan:   consensus.InitRoundUpdate(eventBroker),
		pubKeyBLS:   pubKeyBLS,
		m:           m,
		c:           committee.LaunchStore(eventBroker, db),
		bidList:     bidList,
		transactor:  transactor,
		amount:      amount,
		lockTime:    lockTime,
		offset:      offset,
	}, nil
}

func Launch(eventBroker wire.EventBroker, db database.DB, pubKeyBLS []byte, m ristretto.Scalar, transactor *transactor.Transactor, amount, lockTime, offset uint64) error {
	maintainer, err := newMaintainer(eventBroker, db, pubKeyBLS, m, transactor, amount, lockTime, offset)
	if err != nil {
		return err
	}

	// Let's see if we have active stakes and bids already
	maintainer.bidEndHeight = maintainer.findActiveBid()
	// If not, we create them here
	if maintainer.bidEndHeight == 0 {
		if err := maintainer.sendBid(); err != nil {
			return err
		}
	}

	maintainer.stakeEndHeight = maintainer.findActiveStake()
	if maintainer.stakeEndHeight == 0 {
		if err := maintainer.sendStake(); err != nil {
			return err
		}
	}

	// Set up listening loop
	go maintainer.listen()
	return nil
}

func (m *maintainer) listen() {
	for {
		select {
		case round := <-m.roundChan:
			m.bidList.RemoveExpired(round)
			if round+m.offset >= m.bidEndHeight {
				endHeight := m.findActiveBid()

				// Only send bid if this is the first time we notice it's about to expire
				if endHeight == m.bidEndHeight && m.bidEndHeight != 0 {
					if err := m.sendBid(); err != nil {
						l.WithError(err).Warnln("could not send bid tx")
						continue
					}
					m.bidEndHeight = 0
				} else {
					m.bidEndHeight = endHeight
				}
			}

			if round+m.offset >= m.stakeEndHeight {
				endHeight := m.findActiveStake()

				// Only send stake if this is the first time we notice it's about to expire
				if endHeight == m.stakeEndHeight && m.stakeEndHeight != 0 {
					if err := m.sendStake(); err != nil {
						l.WithError(err).Warnln("could not send stake tx")
						continue
					}
					m.stakeEndHeight = 0
				} else {
					m.stakeEndHeight = endHeight
				}
			}
		case bid := <-m.bidChan:
			m.bidList.AddBid(bid)
		}
	}
}

func (m *maintainer) findActiveBid() uint64 {
	for _, bid := range *m.bidList {
		if bytes.Equal(m.m.Bytes(), bid.M[:]) {
			return bid.EndHeight
		}
	}

	return 0
}

func (m *maintainer) findActiveStake() uint64 {
	p := m.c.Provisioners()
	member := p.GetMember(m.pubKeyBLS)
	if member != nil {
		return member.Stakes[0].EndHeight
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
