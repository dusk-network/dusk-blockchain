package notary_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	n "gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/notary"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

type MockCollector struct{ f func(*bytes.Buffer) error }

func (m *MockCollector) Collect(b *bytes.Buffer) error { return m.f(b) }

func TestSubscriber(t *testing.T) {
	bus := wire.New()
	tbytes, _ := crypto.RandEntropy(32)
	tbuf := bytes.NewBuffer(tbytes)
	collector := &MockCollector{
		func(b *bytes.Buffer) error {
			require.Equal(t, tbuf, t)
			return nil
		},
	}

	sub := n.NewEventSubscriber(bus, collector, "pippo")
	go sub.Accept()

	bus.Publish("pippo", tbuf)

}
