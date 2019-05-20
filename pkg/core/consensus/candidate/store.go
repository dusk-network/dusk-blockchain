package candidate

import (
	"bytes"
	"encoding/hex"

	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

var empty struct{}

type candidateStore struct {
	publisher wire.EventPublisher
	handler   consensus.EventHandler
	store     map[string]wire.Event
}

func newCandidateStore(publisher wire.EventPublisher) *candidateStore {
	return &candidateStore{
		publisher: publisher,
		handler:   newCandidateHandler(),
		store:     make(map[string]wire.Event),
	}
}

func (c *candidateStore) Process(ev wire.Event) {
	cev := ev.(*CandidateEvent)
	cev.SetHash()
	hashStr := hex.EncodeToString(cev.Header.Hash)
	c.store[hashStr] = cev
}

func (c *candidateStore) verifyBlock(hash string) error {
	// TODO: use req-resp communication with chain to verify
	return c.handler.Verify(c.store[hash])
}

func (c *candidateStore) sendWinningBlock(hash string) {
	ev := c.store[hash]
	if ev != nil {
		buf := new(bytes.Buffer)
		cev := ev.(*CandidateEvent)
		if err := cev.Encode(buf); err != nil {
			panic(err)
		}

		message, err := wire.AddTopic(buf, topics.Block)
		if err != nil {
			panic(err)
		}
		c.publisher.Stream(string(topics.Gossip), message)
		log.WithFields(log.Fields{
			"process": "candidate",
			"hash":    hash,
		}).Traceln("block gossiped")
	}

	// empty the store
	c.store = make(map[string]wire.Event)
}
