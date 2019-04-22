package reputation

import (
	"bytes"
	"encoding/hex"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

// MaxStrikes is the maximum allowed amount of strikes in a single round
const maxStrikes uint8 = 2

type moderator struct {
	publisher wire.EventPublisher
	strikes   map[string]uint8

	roundChan    chan uint64
	absenteeChan chan [][]byte
}

// LaunchReputationComponent creates a component that tallies strikes for provisioners.
func LaunchReputationComponent(eventBroker wire.EventBroker) *moderator {
	moderator := newModerator(eventBroker)
	go moderator.listen()
	return moderator
}

func newModerator(eventBroker wire.EventBroker) *moderator {
	return &moderator{
		publisher:    eventBroker,
		strikes:      make(map[string]uint8),
		roundChan:    consensus.InitRoundUpdate(eventBroker),
		absenteeChan: initAbsenteeCollector(eventBroker),
	}
}

func (m *moderator) listen() {
	for {
		select {
		case <-m.roundChan:
			// clean strikes map on round update
			m.strikes = make(map[string]uint8)
		case absentees := <-m.absenteeChan:
			m.addStrikes(absentees...)
		}
	}
}

// Increase the strike count for the `absentees` by one. If the amount of strikes
// exceeds `maxStrikes`, we tell the committee store to remove this provisioner.
func (m *moderator) addStrikes(absentees ...[]byte) {
	for _, absentee := range absentees {
		absenteeStr := hex.EncodeToString(absentee)
		m.strikes[absenteeStr]++
		if m.strikes[absenteeStr] >= maxStrikes {
			m.publisher.Publish(msg.RemoveProvisionerTopic, bytes.NewBuffer(absentee))
		}
	}
}
