package processing

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
)

// Ponger responds to a Ping on a channel
type Ponger struct {
	responseChan chan<- *bytes.Buffer
}

// NewPonger instantiates a new Ponger with a new channel
func NewPonger(responseChan chan<- *bytes.Buffer) Ponger {
	return Ponger{responseChan}
}

// Pong pushes a pong buffer to a channel
func (p *Ponger) Pong() {
	buf := new(bytes.Buffer)
	_ = topics.Prepend(buf, topics.Pong)
	p.responseChan <- buf
}
