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
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database/lite"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

func TestAddProvisioner(t *testing.T) {
	bus := wire.NewEventBus()
	_, db := lite.CreateDBConnection()
	c := launchStore(bus, db)

	newProvisioner(10, bus, 0, 1000)
	// Give the committee store some time to add the provisioner
	time.Sleep(100 * time.Millisecond)

	p := c.Provisioners()
	assert.Equal(t, 1, p.Size(1))
}

func TestRemoveProvisioner(t *testing.T) {
	bus := wire.NewEventBus()
	_, db := lite.CreateDBConnection()
	c := launchStore(bus, db)

	k := newProvisioner(10, bus, 0, 1000)
	// Give the committee store some time to add the provisioner
	time.Sleep(100 * time.Millisecond)

	bus.Publish(msg.RemoveProvisionerTopic, bytes.NewBuffer(k.BLSPubKey.Marshal()))
	// Give the store some time to remove the provisioner
	time.Sleep(100 * time.Millisecond)

	p := c.Provisioners()
	assert.Equal(t, 0, p.Size(1))
}

// Test that a committee cache keeps copies of produced voting committees.
func TestUpsertCommitteeCache(t *testing.T) {
	bus := wire.NewEventBus()
	_, db := lite.CreateDBConnection()
	e := NewExtractor(bus, db)

	// add some provisioners
	newProvisioners(3, 10, bus)
	// give the committee some time to add the provisioners
	time.Sleep(100 * time.Millisecond)

	// run UpsertCommitteCache 4 times, twice on the same state
	_ = e.UpsertCommitteeCache(1, 1, 3)
	_ = e.UpsertCommitteeCache(1, 1, 3)
	_ = e.UpsertCommitteeCache(1, 2, 3)
	_ = e.UpsertCommitteeCache(1, 3, 3)

	// committeeCache should now hold 3 VotingCommittees
	assert.Equal(t, 3, len(e.committeeCache))

	// now run IsMember for another round
	_ = e.UpsertCommitteeCache(2, 1, 3)

	// committeeCache should now hold 1 VotingCommittee
	assert.Equal(t, 1, len(e.committeeCache))
}

// Test that an Extractor clears its committee cache when asked to produce a committee
// for a different round.
func TestCleanCommitteeCache(t *testing.T) {
	bus := wire.NewEventBus()
	_, db := lite.CreateDBConnection()
	e := NewExtractor(bus, db)

	// add some provisioners
	newProvisioners(3, 10, bus)
	// give the committee some time to add the provisioners
	time.Sleep(100 * time.Millisecond)

	// run UpsertCommitteCache once
	_ = e.UpsertCommitteeCache(1, 1, 3)

	// committeeCache should now hold 1 VotingCommittee
	assert.Equal(t, 1, len(e.committeeCache))

	// now run IsMember for another round
	_ = e.UpsertCommitteeCache(2, 1, 3)

	// committeeCache should now hold 1 VotingCommittee
	assert.Equal(t, 1, len(e.committeeCache))
}

func TestRemoveExpired(t *testing.T) {
	bus := wire.NewEventBus()
	_, db := lite.CreateDBConnection()
	e := NewExtractor(bus, db)

	// add some provisioners
	newProvisioners(3, 10, bus)
	// give the committee some time to add the provisioners
	time.Sleep(100 * time.Millisecond)

	// These stakes all expire at height 1000 - so let's remove them
	bs := make([]byte, 8)
	binary.LittleEndian.PutUint64(bs, 1001)
	err := e.RemoveExpiredProvisioners(bytes.NewBuffer(bs))
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), e.totalWeight)

	p := e.Store.Provisioners()
	assert.Equal(t, 0, p.Size(1001))
}

func TestMultiStake(t *testing.T) {
	bus := wire.NewEventBus()
	_, db := lite.CreateDBConnection()
	e := NewExtractor(bus, db)

	// Add a provisioner, and give him 3 stakes
	k := newProvisioner(100, bus, 0, 1000)
	publishNewStake(100, bus, 0, 1500, k)
	publishNewStake(100, bus, 0, 2000, k)

	// Check stats
	p := e.Store.Provisioners()
	assert.Equal(t, 1, p.Size(1))
	assert.Equal(t, uint64(300), e.totalWeight)

	// Remove first stake
	bs := make([]byte, 8)
	binary.LittleEndian.PutUint64(bs, 1001)
	e.RemoveExpiredProvisioners(bytes.NewBuffer(bs))

	// Check stats
	assert.Equal(t, 1, p.Size(1001))
	assert.Equal(t, uint64(200), e.totalWeight)

	// Remove second and third stake
	bs = make([]byte, 8)
	binary.LittleEndian.PutUint64(bs, 2001)
	e.RemoveExpiredProvisioners(bytes.NewBuffer(bs))

	// Check stats
	assert.Equal(t, 0, p.Size(2001))
	assert.Equal(t, uint64(0), e.totalWeight)
}

func newMockEvent(sender []byte) wire.Event {
	mockEvent := &mocks.Event{}
	mockEvent.On("Sender").Return(sender)
	return mockEvent
}

func publishNewStake(stake uint64, eb *wire.EventBus, startHeight, endHeight uint64, k user.Keys) {
	buffer := bytes.NewBuffer(*k.EdPubKey)
	_ = encoding.WriteVarBytes(buffer, k.BLSPubKeyBytes)

	_ = encoding.WriteUint64(buffer, binary.LittleEndian, stake)
	_ = encoding.WriteUint64(buffer, binary.LittleEndian, startHeight)
	_ = encoding.WriteUint64(buffer, binary.LittleEndian, endHeight)

	eb.Publish(msg.NewProvisionerTopic, buffer)
}

func newProvisioner(stake uint64, eb *wire.EventBus, startHeight, endHeight uint64) user.Keys {
	k, _ := user.NewRandKeys()
	publishNewStake(stake, eb, startHeight, endHeight, k)
	return k
}

func newProvisioners(amount int, stake uint64, eb *wire.EventBus) {
	for i := 0; i < amount; i++ {
		_ = newProvisioner(stake, eb, 0, 1000)
	}
}
