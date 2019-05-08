package agreement

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

func TestNotification(t *testing.T) {
	bus := wire.NewEventBus()
	agChan := LaunchNotification(bus)

	hash, _ := crypto.RandEntropy(32)
	keys := make([]*user.Keys, 2)
	for i := 0; i < 2; i++ {
		keys[i], _ = user.NewRandKeys()
	}
	PublishMock(bus, hash, 1, 2, keys)
	msg := <-agChan
	assert.Equal(t, hash, msg.AgreedHash)
}
