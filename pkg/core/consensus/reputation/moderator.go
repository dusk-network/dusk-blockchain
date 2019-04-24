package reputation

import (
	"bytes"
	"encoding/hex"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"

	log "github.com/sirupsen/logrus"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

// MaxStrikes is the maximum allowed amount of strikes in a single round
const maxStrikes uint8 = 3

type moderator struct {
	publisher wire.EventPublisher
	strikes   map[string]uint8

	roundChan    <-chan uint64
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
	exceededMaxStrikes := make([][]byte, 0)
	for _, absentee := range absentees {
		absenteeStr := hex.EncodeToString(absentee)
		m.strikes[absenteeStr]++
		if m.strikes[absenteeStr] >= maxStrikes {
			log.WithFields(log.Fields{
				"process":     "reputation",
				"provisioner": absenteeStr,
			}).Debugln("removing provisioner")
			exceededMaxStrikes = append(exceededMaxStrikes, absentee)
		}
	}

	if len(exceededMaxStrikes) > 0 {
		m.publisher.Publish(msg.RemoveProvisionerTopic, marshalKeys(exceededMaxStrikes...))
	}
}

func marshalKeys(absenteeKeys ...[]byte) *bytes.Buffer {
	buf := new(bytes.Buffer)
	if err := encoding.WriteVarInt(buf, uint64(len(absenteeKeys))); err != nil {
		panic(err)
	}
	for i := 0; i < len(absenteeKeys); i++ {
		if err := encoding.WriteVarBytes(buf, absenteeKeys[i]); err != nil {
			panic(err)
		}
	}

	return buf
}
