package consensus

import (
	"bytes"
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-crypto/bls"
)

type Component interface {
	Finalize()
	Initialize(Signer) ([]Subscriber, EventHandler)
	Notify(topics.Topic, []wire.Event, bool) error
}

type Listener interface {
	Notify(bytes.Buffer, *header.Header) error
}

type CallbackListener struct {
	callback func(bytes.Buffer, *header.Header) error
}

func NewCallbackListener(callback func(bytes.Buffer, *header.Header) error) Listener {
	return &CallbackListener{callback}
}

type Signer struct {
	keys      user.Keys
	consensus *Consensus
}

type Subscriber struct {
	topic    topics.Topic
	listener eventbus.Listener
}

type Consensus struct {
	eventBus    eventbus.Broker
	rpcBus      *rpcbus.RPCBus
	keys        user.Keys
	components  []Component
	lock        sync.RWMutex
	subscribers map[topics.Topic][]Subscriber
}

func (c *Consensus) New(eventBus eventbus.Broker, rpcBus *rpcbus.RPCBus, keys user.Keys) *Consensus {
	// TODO: listen for initialization message
	return *Consensus{
		eventBus:    eventBus,
		rpcBus:      rpcBus,
		keys:        keys,
		components:  make([]Component, 4),
		subscribers: eventbus.NewListenerMap(),
	}
}

func (c *Consensus) CollectRoundUpdate(m bytes.Buffer) error {
	r, err := decodeRoundUpdate(m)
	if err != nil {
		return err
	}

	c.lock.Lock()
	defer c.lock.Unlock()
	for _, component := range c.components {
		c.Finalize()
	}

	_ = c.subscribers.Clear()

	c.components = make([]Component, 4)
	subs, handler := agreement.NewComponent(c.eventBus, c.keys, r.Round, r.P)
}

func (s *Signer) BLSSign(payload []byte) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := encoding.WriteUint64LE(buf, s.consensus.getRound()); err != nil {
		return nil, err
	}

	if err := encoding.WriteUint8(buf, s.consensus.getStep()); err != nil {
		return nil, err
	}

	if _, err := buf.Write(payload); err != nil {
		return nil, err
	}

	signedHash, err := bls.Sign(s.keys.BLSSecretKey, s.keys.BLSPubKey, payload)
	if err != nil {
		return nil, err
	}

	return signedHash.Compress(), nil
}

func (c *Consensus) getRound() uint64 {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.round
}

func (c *Consensus) getStep() uint8 {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.step
}
