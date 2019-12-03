package requesting_test

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/requesting"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

func TestProvideAgreements(t *testing.T) {
	eb := eventbus.New()
	respChan := make(chan *bytes.Buffer, 1)
	a := requesting.NewAgreementCache(eb, respChan)
	hash, _ := crypto.RandEntropy(32)
	p, keys := consensus.MockProvisioners(10)
	agreements := make([]*bytes.Buffer, 0, 10)
	for i := 0; i < 10; i++ {
		agreements = append(agreements, agreement.MockAgreement(hash, 1, 3, keys, p, i))
	}

	// Concatenate buffers
	buf := new(bytes.Buffer)
	if err := encoding.WriteUint64LE(buf, 20); err != nil {
		t.Fatal(err)
	}

	if err := encoding.WriteVarInt(buf, uint64(len(agreements))); err != nil {
		t.Fatal(err)
	}

	for _, b := range agreements {
		buf.ReadFrom(b)
	}

	// Send agreements to the cache
	eb.Publish(topics.AddAgreements, buf)

	// Now request them
	reqBuf := new(bytes.Buffer)
	if err := encoding.WriteUint64LE(reqBuf, 20); err != nil {
		t.Fatal(err)
	}

	assert.NoError(t, a.SendAgreements(reqBuf))

	resp := <-respChan
	assert.Equal(t, buf.Bytes()[8:], resp.Bytes())
}
