package voting

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/events"
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

	//eliminating topic
	topic := make([]byte, 15)
	_, _ = signed.Read(topic)

	// removing the ED signature
	sig := make([]byte, 64)
	assert.NoError(t, encoding.Read512(signed, &sig))
	assert.True(t, ed25519.Verify(*k.EdPubKey, signed.Bytes(), sig))

	ev, err := events.NewReductionUnMarshaller().Deserialize(signed)
	assert.NoError(t, err)
	r := ev.(*events.Reduction)

	// testing that the BLS Public Key has been properly added
	assert.Equal(t, k.BLSPubKeyBytes, r.PubKeyBLS)
	// testing the fields of the reduction message
	assert.Equal(t, uint64(1), r.Round)
	assert.Equal(t, uint8(1), r.Step)
	assert.Equal(t, hash, r.BlockHash)
	// testing the size of the BLS compressed signature
	assert.Equal(t, 33, len(r.SignedHash))

	//testing the validity of the signature
	b := new(bytes.Buffer)
	assert.NoError(t, events.MarshalSignableVote(b, r.Header))
	assert.NoError(t, msg.VerifyBLSSignature(r.PubKeyBLS, b.Bytes(), r.SignedHash))
}
