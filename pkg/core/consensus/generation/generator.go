package generation

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/marshalling"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-wallet/block"
)

var _ consensus.Component = (*Generator)(nil)

var emptyHash [32]byte

// Generator is the component that signals a restart of score generation and selection
// after a Restart Event is detected
type Generator struct {
	signer      consensus.Signer
	restartID   uint32
	certificate *block.Certificate
}

// NewComponent instantiates a new Generator
func NewComponent() *Generator {
	return &Generator{}
}

// Initialize the Generator by subscribing to `Restart` topic. The Listener is
// marked as LowPriority to allow for the `Selector` to be notified first
func (g *Generator) Initialize(eventPlayer consensus.EventPlayer, signer consensus.Signer, rs consensus.RoundState) []consensus.TopicListener {
	g.signer = signer
	g.certificate = &rs.Certificate

	restartListener := consensus.TopicListener{
		Topic:    topics.Restart,
		Listener: consensus.NewSimpleListener(g.Collect, consensus.LowPriority, false),
	}
	g.restartID = restartListener.Listener.ID()

	return []consensus.TopicListener{restartListener}
}

// Finalize as part of the Component interface
func (g *Generator) Finalize() {}

// ID as part of the Component interface
func (g *Generator) ID() uint32 {
	return g.restartID
}

// Collect `Restart` events and triggers a Generation event
func (g *Generator) Collect(ev consensus.Event) error {
	if g.certificate != nil {
		buf := new(bytes.Buffer)
		if err := marshalling.MarshalCertificate(buf, g.certificate); err != nil {
			return err
		}

		return g.signer.SendWithHeader(topics.Generation, emptyHash[:], buf, g.ID())
	}

	return nil
}
