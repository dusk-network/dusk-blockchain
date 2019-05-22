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
	c := LaunchCommitteeStore(bus, user.Keys{})

	newProvisioner(10, bus)
	// Give the committee store some time to add the provisioner
	time.Sleep(100 * time.Millisecond)

	p := c.copyProvisioners()
	assert.Equal(t, 1, p.Size())
}

func TestRemoveProvisioner(t *testing.T) {
	bus := wire.NewEventBus()
	c := LaunchCommitteeStore(bus, user.Keys{})

	k := newProvisioner(10, bus)
	// Give the committee store some time to add the provisioner
	time.Sleep(100 * time.Millisecond)

	bus.Publish(msg.RemoveProvisionerTopic, bytes.NewBuffer(k.BLSPubKey.Marshal()))
	// Give the store some time to remove the provisioner
	time.Sleep(100 * time.Millisecond)

	p := c.copyProvisioners()
	assert.Equal(t, 0, p.Size())
}

func TestReportAbsentees(t *testing.T) {
	bus := wire.NewEventBus()
	c := LaunchCommitteeStore(bus, user.Keys{})
	e := NewCommitteeExtractor(c, ReductionCommitteeSize)
	absenteesChan := make(chan *bytes.Buffer, 1)
	bus.Subscribe(msg.AbsenteesTopic, absenteesChan)

	k1 := newProvisioner(10, bus)
	k2 := newProvisioner(10, bus)
	k3 := newProvisioner(10, bus)
	// give the committee some time to add the provisioners
	time.Sleep(100 * time.Millisecond)

	// make events
	ev1 := newMockEvent(k1.BLSPubKeyBytes)
	ev2 := newMockEvent(k2.BLSPubKeyBytes)

	evs := []wire.Event{ev1, ev2}

	_ = e.ReportAbsentees(evs, 1, 1)
	absentees := <-absenteesChan
	// absentees should contain the bls pub key of k3
	assert.True(t, bytes.Contains(absentees.Bytes(), k3.BLSPubKeyBytes))
}

func TestUpsertCommitteeCache(t *testing.T) {
	bus := wire.NewEventBus()
	c := LaunchCommitteeStore(bus, user.Keys{})
	e := NewCommitteeExtractor(c, ReductionCommitteeSize)

	// add some provisioners
	k1 := newProvisioner(10, bus)
	_ = newProvisioner(10, bus)
	_ = newProvisioner(10, bus)
	// give the committee some time to add the provisioners
	time.Sleep(100 * time.Millisecond)

	// run IsMember, which should trigger a voting committee creation
	_ = e.IsMember(k1.BLSPubKey.Marshal(), 1, 1)

	// committeeCache should now hold one VotingCommittee
	assert.Equal(t, 1, len(e.committeeCache))
}

func TestCleanCommitteeCache(t *testing.T) {
	bus := wire.NewEventBus()
	c := LaunchCommitteeStore(bus, user.Keys{})
	e := NewCommitteeExtractor(c, ReductionCommitteeSize)

	// add some provisioners
	k1 := newProvisioner(10, bus)
	_ = newProvisioner(10, bus)
	_ = newProvisioner(10, bus)
	// give the committee some time to add the provisioners
	time.Sleep(100 * time.Millisecond)

	// run IsMember a few times
	_ = e.IsMember(k1.BLSPubKey.Marshal(), 1, 1)
	_ = e.IsMember(k1.BLSPubKey.Marshal(), 1, 2)
	_ = e.IsMember(k1.BLSPubKey.Marshal(), 1, 3)

	// committeeCache should now hold 3 VotingCommittees
	assert.Equal(t, 3, len(e.committeeCache))

	// now run IsMember for another round
	_ = e.IsMember(k1.BLSPubKey.Marshal(), 2, 1)

	// committeeCache should now hold 1 VotingCommittee
	assert.Equal(t, 1, len(e.committeeCache))
}

func newMockEvent(sender []byte) wire.Event {
	mockEvent := &mocks.Event{}
	mockEvent.On("Sender").Return(sender)
	return mockEvent
}

func newProvisioner(amount uint64, eb *wire.EventBus) user.Keys {
	k, _ := user.NewRandKeys()
	buffer := bytes.NewBuffer(*k.EdPubKey)
	_ = encoding.WriteVarBytes(buffer, k.BLSPubKeyBytes)

	_ = encoding.WriteUint64(buffer, binary.LittleEndian, amount)

	eb.Publish(msg.NewProvisionerTopic, buffer)
	return k
}
