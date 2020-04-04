package processing

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
)

type Ponger struct {
	responseChan chan<- *bytes.Buffer
}

func NewPonger(responseChan chan<- *bytes.Buffer) Ponger {
	return Ponger{responseChan}
}

func (p *Ponger) Pong() {
	buf := new(bytes.Buffer)
	_ = topics.Prepend(buf, topics.Pong)
	p.responseChan <- buf
}
