package reputation

import (
	"bytes"
	"encoding/hex"
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/msg"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"

	log "github.com/sirupsen/logrus"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
)

// MaxStrikes is the maximum allowed amount of strikes in a single round
const MaxStrikes uint8 = 6

// Filter is capable of filtering out a set of absent nodes, given a set of votes,
// a round, and a step.
type Filter interface {
	FilterAbsentees([]wire.Event, uint64, uint8) user.VotingCommittee
}

type moderator struct {
	publisher eventbus.Publisher
	lock      sync.RWMutex
	strikes   map[string]uint8
	round     uint64

	roundChan <-chan uint64
}

// Launch creates a component that tallies strikes for provisioners.
func Launch(eventBroker eventbus.Broker) {
	moderator := newModerator(eventBroker)
	eventBroker.SubscribeCallback(msg.AbsenteesTopic, moderator.strikeAbsentees)
	go moderator.listen()
}

func newModerator(eventBroker eventbus.Broker) *moderator {
	return &moderator{
		publisher: eventBroker,
		strikes:   make(map[string]uint8),
		roundChan: consensus.InitRoundUpdate(eventBroker),
	}
}

// Listen for round updates.
func (m *moderator) listen() {
	for {
		select {
		case round := <-m.roundChan:
			// clean strikes map on round update
			m.lock.Lock()
			m.strikes = make(map[string]uint8)
			m.round = round
			m.lock.Unlock()
		}
	}
}

func (m *moderator) strikeAbsentees(b *bytes.Buffer) error {
	absentees, err := decodeAbsentees(b)
	if err != nil {
		return err
	}

	m.lock.Lock()
	defer m.lock.Unlock()
	if absentees.round == m.round {
		log.WithFields(log.Fields{
			"process": "reputation",
			"round":   m.round,
		}).Debugln("striking absentees")
		m.addStrikes(absentees.pks)
	}

	return nil
}

// Increase the strike count for the `absentee` by one. If the amount of strikes
// exceeds `maxStrikes`, we tell the committee store to remove this provisioner.
func (m *moderator) addStrikes(pks [][]byte) {
	for _, pk := range pks {
		absenteeStr := hex.EncodeToString(pk)
		m.strikes[absenteeStr]++
		if m.strikes[absenteeStr] >= MaxStrikes {
			log.WithFields(log.Fields{
				"process":     "reputation",
				"provisioner": absenteeStr,
			}).Debugln("removing provisioner")
			m.publisher.Publish(msg.RemoveProvisionerTopic, bytes.NewBuffer(pk))
		}
	}
}
