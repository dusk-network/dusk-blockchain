package committee

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/mocks"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

func TestAddProvisioner(t *testing.T) {
	bus := wire.NewEventBus()
	c := LaunchCommitteeStore(bus, nil)

	newProvisioner(10, bus)
	// Give the committee store some time to add the provisioner
	time.Sleep(100 * time.Millisecond)

	p := c.copyProvisioners()
	assert.Equal(t, 1, p.VotingCommitteeSize())
}

func TestRemoveProvisioner(t *testing.T) {
	bus := wire.NewEventBus()
	c := LaunchCommitteeStore(bus, nil)

	k := newProvisioner(10, bus)
	// Give the committee store some time to add the provisioner
	time.Sleep(100 * time.Millisecond)

	bus.Publish(msg.RemoveProvisionerTopic, bytes.NewBuffer(k.BLSPubKey.Marshal()))
	// Give the store some time to remove the provisioner
	time.Sleep(100 * time.Millisecond)

	p := c.copyProvisioners()
	assert.Equal(t, 0, p.VotingCommitteeSize())
}

func TestReportAbsentees(t *testing.T) {
	bus := wire.NewEventBus()
	c := LaunchCommitteeStore(bus, nil)
	absenteesChan := make(chan *bytes.Buffer, 1)
	bus.Subscribe(msg.AbsenteesTopic, absenteesChan)

	k1 := newProvisioner(10, bus)
	k2 := newProvisioner(10, bus)
	k3 := newProvisioner(10, bus)
	// give the committee some time to add the provisioners
	time.Sleep(100 * time.Millisecond)

	// make events
	ev1 := newMockEvent(k1.BLSPubKey.Marshal())
	ev2 := newMockEvent(k2.BLSPubKey.Marshal())

	evs := []wire.Event{ev1, ev2}

	_ = c.ReportAbsentees(evs, 1, 1)
	absentees := <-absenteesChan
	// absentees should contain the bls pub key of k3
	assert.True(t, bytes.Contains(absentees.Bytes(), k3.BLSPubKey.Marshal()))
}

func newMockEvent(sender []byte) wire.Event {
	mockEvent := &mocks.Event{}
	mockEvent.On("Sender").Return(sender)
	return mockEvent
}

func newProvisioner(amount uint64, eb *wire.EventBus) *user.Keys {
	k, _ := user.NewRandKeys()
	buffer := bytes.NewBuffer(*k.EdPubKey)
	_ = encoding.WriteVarBytes(buffer, k.BLSPubKey.Marshal())

	_ = encoding.WriteUint64(buffer, binary.LittleEndian, amount)

	eb.Publish(msg.NewProvisionerTopic, buffer)
	return k
}
