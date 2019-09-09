package consensus

import (
	"bytes"
	"errors"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/committee"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	"github.com/dusk-network/dusk-blockchain/pkg/core/transactor"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/wallet/transactions"
)

// The maintainer is a process that keeps note of when certain consensus transactions
// expire, and makes sure the node remains within the bidlist/committee, when those
// transactions are close to expiring.
type maintainer struct {
	eventBroker wire.EventBroker
	roundChan   <-chan uint64
	bidChan     <-chan Bid

	pubKeyBLS []byte
	m         ristretto.Scalar
	c         *committee.Store
	bidList   *BidList

	bidEndHeight   uint64
	stakeEndHeight uint64

	value, lockTime uint64

	db         database.DB
	transactor *transactor.Transactor
}

func newMaintainer(eventBroker wire.EventBroker, pubKeyBLS []byte, m ristretto.Scalar, transactor *transactor.Transactor) (*maintainer, error) {
	_, db := heavy.CreateDBConnection()
	// bidList, err := NewBidList(db)
	// if err != nil {
	// 	log.WithFields(log.Fields{
	// 		"process": "maintainer",
	// 		"error":   err,
	// 	}).Errorln("could not repopulate bidlist")
	// 	return nil, err
	// }

	return &maintainer{
		eventBroker: eventBroker,
		roundChan:   InitRoundUpdate(eventBroker),
		pubKeyBLS:   pubKeyBLS,
		m:           m,
		c:           committee.LaunchStore(eventBroker, db),
		// bidList:     bidList,
		db:         db,
		transactor: transactor,
	}, nil
}

func LaunchMaintainer(eventBroker wire.EventBroker, pubKeyBLS []byte, m ristretto.Scalar, transactor *transactor.Transactor) error {
	maintainer, err := newMaintainer(eventBroker, pubKeyBLS, m, transactor)
	if err != nil {
		return err
	}

	// Let's see if we have active stakes and bids already
	maintainer.bidEndHeight, err = maintainer.findActiveBid()
	if err != nil {
		return err
	}

	maintainer.stakeEndHeight = maintainer.findActiveStake()
	go maintainer.listen()
	return nil
}

func (m *maintainer) listen() {
	for {
		round := <-m.roundChan
		if round >= m.bidEndHeight {

		}

		if round >= m.stakeEndHeight {
			endHeight := m.findActiveStake()
			if endHeight != 0 {
				// Send stake and wait for it to get included

			}
			m.stakeEndHeight = endHeight
		}
	}
}

func (m *maintainer) findActiveBid() (uint64, error) {
	return 0, nil
}

func (m *maintainer) findActiveStake() uint64 {
	p := m.c.Provisioners()
	member := p.GetMember(m.pubKeyBLS)
	if member != nil {
		return member.Stakes[0].EndHeight
	}

	return 0
}

// TODO: DRY
func (m *maintainer) sendBid() error {
	balance, err := m.transactor.Balance()
	if err != nil {
		return err
	}

	if balance < float64(m.value) {
		return errors.New("insufficient balance for automated bid tx")
	}

	bid, err := m.transactor.CreateBidTx(m.value, m.lockTime)
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
	balance, err := m.transactor.Balance()
	if err != nil {
		return err
	}

	if balance < float64(m.value) {
		return errors.New("insufficient balance for automated stake tx")
	}

	stake, err := m.transactor.CreateStakeTx(m.value, m.lockTime)
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
