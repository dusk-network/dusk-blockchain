package responding

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
)

// ProcessPing will simply return a Pong message.
// Satisfies the peer.ProcessorFunc interface.
func ProcessPing(_ message.Message) ([]*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	if err := topics.Prepend(buf, topics.Pong); err != nil {
		return nil, err
	}

	return []*bytes.Buffer{buf}, nil
}
