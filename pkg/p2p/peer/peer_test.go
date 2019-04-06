package peer

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

func TestDupeFilter(t *testing.T) {
	test := bytes.NewBufferString("This is a test")

	eventbus := wire.New()
	dupeMap := newDupeMap(eventbus)
	go dupeMap.cleanOnRound()

	assert.True(t, dupeMap.canFwd(test))
	assert.False(t, dupeMap.canFwd(test))

	publishRound(eventbus, 2)
	assert.False(t, dupeMap.canFwd(test))
}

func TestDupeFilterCleanup(t *testing.T) {
	test := bytes.NewBufferString("This is a test")

	eventbus := wire.New()
	dupeMap := newDupeMap(eventbus)
	go dupeMap.cleanOnRound()

	assert.True(t, dupeMap.canFwd(test))
	assert.False(t, dupeMap.canFwd(test))

	publishRound(eventbus, 4)
	assert.True(t, dupeMap.canFwd(test))
	assert.False(t, dupeMap.canFwd(test))

	publishRound(eventbus, 5)
	assert.False(t, dupeMap.canFwd(test))
}

func publishRound(eventbus *wire.EventBus, round uint64) {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, round)
	eventbus.Publish(msg.RoundUpdateTopic, bytes.NewBuffer(b))
	<-time.After(30 * time.Millisecond)
}
