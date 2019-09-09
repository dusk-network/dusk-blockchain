package maintainer

import (
	"bytes"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/committee"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/generation"
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

	pubKeyBLS    []byte
	m            ristretto.Scalar
	c            *committee.Store
	bidRetriever *generation.BidRetriever

	bidEndHeight   uint64
	stakeEndHeight uint64

	amount, lockTime, offset uint64

	transactor *transactor.Transactor
}

func newMaintainer(eventBroker wire.EventBroker, db database.DB, pubKeyBLS []byte, m ristretto.Scalar, transactor *transactor.Transactor, amount, lockTime, offset uint64) (*maintainer, error) {
	if db == nil {
		_, db = heavy.CreateDBConnection()
	}
	return &maintainer{
		eventBroker:  eventBroker,
		roundChan:    consensus.InitRoundUpdate(eventBroker),
		pubKeyBLS:    pubKeyBLS,
		m:            m,
		c:            committee.LaunchStore(eventBroker, db),
		bidRetriever: generation.NewBidRetriever(nil),
		transactor:   transactor,
		amount:       amount,
		lockTime:     lockTime,
		offset:       offset,
	}, nil
}

func Launch(eventBroker wire.EventBroker, db database.DB, pubKeyBLS []byte, m ristretto.Scalar, transactor *transactor.Transactor, amount, lockTime, offset uint64) error {
	maintainer, err := newMaintainer(eventBroker, db, pubKeyBLS, m, transactor, amount, lockTime, offset)
	if err != nil {
		return err
	}

	// Let's see if we have active stakes and bids already
	maintainer.bidEndHeight, err = maintainer.findActiveBid()
	if err != nil && err != generation.NoBidFound {
		return err
	}

	maintainer.stakeEndHeight = maintainer.findActiveStake()

	// Set up listening loop
	go maintainer.listen()
	return nil
}

func (m *maintainer) listen() {
	for {
		round := <-m.roundChan
		if round+m.offset >= m.bidEndHeight {
			endHeight, err := m.findActiveBid()
			if err != nil && err != generation.NoBidFound {
				// If we get a more serious error, we exit the loop. It means there is something wrong with the db.
				l.WithError(err).Errorln("problem attempting to search for bids")
				return
			}

			if endHeight == 0 {
				if err := m.sendBid(); err != nil {
					l.WithError(err).Warnln("could not send bid tx")
					continue
				}
			}

			m.bidEndHeight = endHeight
		}

		if round+m.offset >= m.stakeEndHeight {
			endHeight := m.findActiveStake()
			if endHeight == 0 {
				if err := m.sendStake(); err != nil {
					l.WithError(err).Warnln("could not send stake tx")
					continue
				}
			}

			m.stakeEndHeight = endHeight
		}
	}
}

func (m *maintainer) findActiveBid() (uint64, error) {
	_, height, err := m.bidRetriever.SearchForBid(m.m.Bytes())
	if err != nil {
		return 0, err
	}

	return height, nil
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
