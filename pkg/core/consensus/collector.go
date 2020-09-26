package consensus

import (
	"bytes"
	"context"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message/payload"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-crypto/bls"
)

// roundCollector is a simple wrapper over a channel to get round notifications.
// It is not supposed to be used directly. Components interestesd in Round updates
// should use InitRoundUpdate instead
type (
	roundCollector struct {
		roundChan chan RoundUpdate
	}

	acceptedBlockCollector struct {
		blockChan chan<- block.Block
	}

	// RoundUpdate carries the data about the new Round, such as the active
	// Provisioners, the BidList, the Seed and the Hash
	RoundUpdate struct {
		Round uint64
		P     user.Provisioners
		Seed  []byte
		Hash  []byte
	}

	// Phase is used whenever an instantiation is needed.
	Phase interface {
		// Fn accepts as an
		// argument an interface, usually a message or the result  of the state
		// function execution. It provides the capability to create a closure of sort
		Fn(interface{}) PhaseFn
	}

	// PhaseFn represents the recursive consensus state function
	PhaseFn func(context.Context, *Queue, chan message.Message, RoundUpdate, uint8) (PhaseFn, error)

	// Emitter is a simple struct to pass the communication channels that the steps should be
	// able to emit onto
	Emitter struct {
		EventBus    *eventbus.EventBus
		RPCBus      *rpcbus.RPCBus
		Keys        key.Keys
		PubkeyBuf   bytes.Buffer
		Proxy       transactions.Proxy
		PubKey      *transactions.PublicKey
		TimerLength time.Duration
	}
)

// Sign a header
func (e *Emitter) Sign(h header.Header) ([]byte, error) {
	preimage := new(bytes.Buffer)
	if err := header.MarshalSignableVote(preimage, h); err != nil {
		return nil, err
	}

	signedHash, err := bls.Sign(e.Keys.BLSSecretKey, e.Keys.BLSPubKey, preimage.Bytes())
	if err != nil {
		return nil, err
	}

	return signedHash.Compress(), nil
}

// Gossip concatenates the topic, the header and the payload,
// and gossips it to the rest of the network.
func (e *Emitter) Gossip(msg message.Message) error {
	// message.Marshal takes care of prepending the topic, marshaling the
	// header, etc
	buf, err := message.Marshal(msg)
	if err != nil {
		return err
	}

	serialized := message.New(msg.Category(), buf)

	// gossip away
	_ = e.EventBus.Publish(topics.Gossip, serialized)
	return nil
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers
func (r RoundUpdate) Copy() payload.Safe {
	ru := RoundUpdate{
		Round: r.Round,
		P:     r.P.Copy(),
		Seed:  make([]byte, len(r.Seed)),
		Hash:  make([]byte, len(r.Hash)),
	}

	copy(ru.Seed, r.Seed)
	copy(ru.Hash, r.Hash)
	return ru
}

// InitRoundUpdate initializes a Round update channel and fires up the TopicListener
// as well. Its purpose is to lighten up a bit the amount of arguments in creating
// the handler for the collectors. Also it removes the need to store subscribers on
// the consensus process
func InitRoundUpdate(subscriber eventbus.Subscriber) <-chan RoundUpdate {
	roundChan := make(chan RoundUpdate, 1)
	roundCollector := &roundCollector{roundChan}
	collectListener := eventbus.NewCallbackListener(roundCollector.Collect)
	if config.Get().General.SafeCallbackListener {
		collectListener = eventbus.NewSafeCallbackListener(roundCollector.Collect)
	}
	subscriber.Subscribe(topics.RoundUpdate, collectListener)
	return roundChan
}

// Collect as specified in the EventCollector interface. In this case Collect simply
// performs unmarshaling of the round event
func (r *roundCollector) Collect(m message.Message) {
	r.roundChan <- m.Payload().(RoundUpdate)
}

// InitAcceptedBlockUpdate init listener to get updates about lastly accepted block in the chain
func InitAcceptedBlockUpdate(subscriber eventbus.Subscriber) (chan block.Block, uint32) {
	acceptedBlockChan := make(chan block.Block)
	collector := &acceptedBlockCollector{acceptedBlockChan}
	collectListener := eventbus.NewCallbackListener(collector.Collect)
	if config.Get().General.SafeCallbackListener {
		collectListener = eventbus.NewSafeCallbackListener(collector.Collect)
	}
	id := subscriber.Subscribe(topics.AcceptedBlock, collectListener)
	return acceptedBlockChan, id
}

// Collect as defined in the EventCollector interface. It reconstructs the bidList and notifies about it
func (c *acceptedBlockCollector) Collect(m message.Message) {
	c.blockChan <- m.Payload().(block.Block)
}
