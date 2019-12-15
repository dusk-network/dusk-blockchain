package consensus_test

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/stretchr/testify/assert"
)

func TestRepublisherIntegrity(t *testing.T) {
	bus := eventbus.New()
	buf := bytes.NewBuffer([]byte{1, 2, 3})
	r := consensus.NewRepublisher(bus, topics.Reduction)
	r.Process(buf)

	assert.Equal(t, []byte{1, 2, 3}, buf.Bytes())
}
