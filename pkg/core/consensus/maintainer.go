package consensus

import (
	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/committee"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
)

// The maintainer is a process that keeps note of when certain consensus transactions
// expire, and makes sure the node remains within the bidlist/committee, when those
// transactions are close to expiring.
type maintainer struct {
	eventBroker    wire.EventBroker
	roundChan      <-chan uint64
	pubKeyBLS      []byte
	k              ristretto.Scalar
	bidEndHeight   uint64
	stakeEndHeight uint64
}

func newMaintainer(eventBroker wire.EventBroker, pubKeyBLS []byte, k ristretto.Scalar) *maintainer {
	return &maintainer{
		eventBroker: eventBroker,
		roundChan:   InitRoundUpdate(eventBroker),
		pubKeyBLS:   pubKeyBLS,
		k:           k,
	}
}

func LaunchMaintainer(eventBroker wire.EventBroker, pubKeyBLS []byte, k ristretto.Scalar) error {
	m := newMaintainer(eventBroker, pubKeyBLS, k)
	// Let's see if we have active stakes and bids already
	_, db := heavy.CreateDBConnection()
	var err error
	m.bidEndHeight, err = m.findActiveBid(db)
	if err != nil {
		return err
	}

	m.stakeEndHeight = m.findActiveStake(db)

	go m.listen()
	return nil
}

func (m *maintainer) listen() {
	for {
		<-m.roundChan
	}
}

func (m *maintainer) findActiveBid(db database.DB) (uint64, error) {
	return 0, nil
}

func (m *maintainer) findActiveStake(db database.DB) uint64 {
	c := committee.LaunchStore(m.eventBroker, db)
	p := c.Provisioners()
	member := p.GetMember(m.pubKeyBLS)
	if member != nil {
		return member.Stakes[0].EndHeight
	}

	return 0
}
