package voting

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/reduction"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
	"golang.org/x/crypto/ed25519"
)

func TestReductionSigning(t *testing.T) {

	hash, _ := crypto.RandEntropy(32)
	bus := wire.NewEventBus()
	msgChan := make(chan *bytes.Buffer)
	bus.Subscribe(string(topics.Gossip), msgChan)

	k, _ := user.NewRandKeys()

	buf := reduction.MockOutgoingReductionBuf(hash, 1, 1)

	LaunchVotingComponent(bus, nil, k)

	<-time.After(200 * time.Millisecond)
	bus.Publish(msg.OutgoingBlockReductionTopic, buf)

	signed := <-msgChan

	//topic
	topic := make([]byte, 15)
	_, _ = signed.Read(topic)

	sig := make([]byte, 64)
	assert.NoError(t, encoding.Read512(signed, &sig))

	edPubKey := make([]byte, 32)
	assert.NoError(t, encoding.Read256(buf, &edPubKey))

	assert.True(t, ed25519.Verify(*k.EdPubKey, signed.Bytes(), sig))
}
