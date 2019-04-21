package committee

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

func TestAddProvisioner(t *testing.T) {
	bus := wire.NewEventBus()
	provisionerChan := make(chan *bytes.Buffer)
	_ = bus.Subscribe(msg.ProvisionerAddedTopic, provisionerChan)
	LaunchCommitteeStore(bus)

	b := newProvisioner(10, nil)
	bus.Publish(msg.NewProvisionerTopic, b)

	newP := <-provisionerChan
	_, _, amount, err := decodeNewProvisioner(newP)
	assert.NoError(t, err)
	assert.Equal(t, uint64(10), amount)
}

func newProvisioner(amount uint64, k *user.Keys) *bytes.Buffer {
	if k == nil {
		k, _ = user.NewRandKeys()
	}
	buffer := bytes.NewBuffer(*k.EdPubKey)
	_ = encoding.WriteVarBytes(buffer, k.BLSPubKey.Marshal())

	_ = encoding.WriteUint64(buffer, binary.LittleEndian, amount)

	return buffer
}
