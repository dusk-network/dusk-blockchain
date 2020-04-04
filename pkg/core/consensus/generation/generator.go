package generation

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
)

var _ consensus.Component = (*Generator)(nil)

//nolint:deadcode
var emptyHash [32]byte

//nolint:deadcode
var emptyPayload = new(bytes.Buffer)

//nolint:deadcode
var genMessage = message.New(topics.Generation, nil)

var restartFactory = consensus.Restarter{}

// Generator is the component that signals a restart of score generation and selection
// after a Restart Event is detected
type Generator struct {
	signer    consensus.Signer
	restartID uint32
}

// NewComponent instantiates a new Generator
func NewComponent() *Generator {
	return &Generator{}
}

// Initialize the Generator by subscribing to `Restart` topic. The Listener is
// marked as LowPriority to allow for the `Selector` to be notified first
func (g *Generator) Initialize(eventPlayer consensus.EventPlayer, signer consensus.Signer, ru consensus.RoundUpdate) []consensus.TopicListener {
	g.signer = signer

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
func (g *Generator) Collect(_ consensus.InternalPacket) error {
	packet := g.signer.Compose(restartFactory)
	msg := message.New(topics.Generation, packet)
	return g.signer.SendInternally(topics.Generation, msg, g.ID())
}
