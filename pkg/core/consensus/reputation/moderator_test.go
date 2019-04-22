package reputation

import (
	"bytes"
	"testing"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

// This test assures proper functionality of adding strikes to a certain
// committee member, up to the maxStrikes count.
func TestStrikes(t *testing.T) {
	eventBus, _, removeProvisionerChan := launchModerator()

	// Send enough strikes for one person so we receive something on removeProvisionerChan
	node, _ := crypto.RandEntropy(32)
	for i := uint8(0); i < maxStrikes; i++ {
		eventBus.Publish(msg.AbsenteesTopic, newAbsenteeBuffer(node))
	}

	// We should now receive the public key of the provisioner who has exceeded maxStrikes
	offenderBuf := <-removeProvisionerChan
	assert.Equal(t, node, offenderBuf.Bytes())
}

// This test assures proper behaviour of the `offenders` map on a round update.
func TestClean(t *testing.T) {
	eventBus, moderator, removeProvisionerChan := launchModerator()

	// Add a strike
	node, _ := crypto.RandEntropy(32)
	eventBus.Publish(msg.AbsenteesTopic, newAbsenteeBuffer(node))
	// wait a bit for the referee to strike...
	time.Sleep(time.Millisecond * 100)

	// Update round
	moderator.roundChan <- 2
	// wait a bit for the referee to update...
	time.Sleep(time.Millisecond * 100)
	// send maxStrikes-1 strikes
	for i := uint8(0); i < maxStrikes-1; i++ {
		eventBus.Publish(msg.AbsenteesTopic, newAbsenteeBuffer(node))
	}

	// check if we get anything on removeProvisionerChan
	timer := time.After(time.Millisecond * 100)
	select {
	case <-removeProvisionerChan:
		assert.Fail(t, "should not have exceeded maxStrikes for the node")
	case <-timer:
		// success
	}
}

func newAbsenteeBuffer(node []byte) *bytes.Buffer {
	buf := new(bytes.Buffer)
	if err := encoding.WriteVarInt(buf, 1); err != nil {
		panic(err)
	}

	if err := encoding.WriteVarBytes(buf, node); err != nil {
		panic(err)
	}

	return buf
}

func launchModerator() (wire.EventBroker, *moderator, chan *bytes.Buffer) {
	eventBus := wire.NewEventBus()
	referee := LaunchReputationComponent(eventBus)
	removeProvisionerChan := make(chan *bytes.Buffer, 1)
	eventBus.Subscribe(msg.RemoveProvisionerTopic, removeProvisionerChan)
	return eventBus, referee, removeProvisionerChan
}
