package generation

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
)

var _ consensus.Component = (*Generator)(nil)

var emptyHash [32]byte
var emptyPayload = new(bytes.Buffer)

type Generator struct {
	signer    consensus.Signer
	restartID uint32
}

func NewComponent() *Generator {
	return &Generator{}
}

func (g *Generator) Initialize(eventPlayer consensus.EventPlayer, signer consensus.Signer, ru consensus.RoundUpdate) []consensus.TopicListener {
	g.signer = signer

	restartListener := consensus.TopicListener{
		Topic:    topics.Restart,
		Listener: consensus.NewSimpleListener(g.Collect, consensus.LowPriority, false),
	}
	g.restartID = restartListener.Listener.ID()

	return []consensus.TopicListener{restartListener}
}

func (g *Generator) Finalize() {}

func (g *Generator) ID() uint32 {
	return g.restartID
}

func (g *Generator) Collect(ev consensus.Event) error {
	return g.signer.SendWithHeader(topics.Generation, emptyHash[:], emptyPayload, g.ID())
}
