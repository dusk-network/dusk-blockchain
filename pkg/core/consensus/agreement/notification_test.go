package agreement

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

func TestNotification(t *testing.T) {
	bus := wire.NewEventBus()
	agChan := LaunchNotification(bus)

	hash, _ := crypto.RandEntropy(32)
	PublishMock(bus, hash, 1, 1, 2)
	msg := <-agChan
	assert.Equal(t, hash, msg.AgreedHash)
}
