package consensus

import (
	"bytes"
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
)

type Consensus struct {
	*SyncState
	eventBus    *eventbus.EventBus
	rpcBus      *rpcbus.RPCBus
	keys        user.Keys
	factories   []ComponentFactory
	components  []Component
	lock        sync.RWMutex
	subscribers map[topics.Topic][]Listener
}

func New(eventBus *eventbus.EventBus, rpcBus *rpcbus.RPCBus, keys user.Keys, factories []ComponentFactory) *Consensus {
	c := &Consensus{
		SyncState:   NewState(),
		eventBus:    eventBus,
		rpcBus:      rpcBus,
		keys:        keys,
		factories:   factories,
		components:  make([]Component, len(factories)),
		subscribers: make(map[topics.Topic][]Listener),
	}

	listener := eventbus.NewCallbackListener(c.CollectEvent)
	eventBus.SubscribeDefault(listener)

	// TODO: listen for initialization message and call reinstantiateComponents()
	return c
}

func (c *Consensus) initialize(component Component, fromScratch bool) Component {
	subs := component.Initialize(c.requestStepUpdate)
	for _, sub := range subs {
		if fromScratch {
			c.eventBus.AddDefaultTopic(sub.topic)
			c.eventBus.Register(sub.topic, NewRepublisher(c.eventBus, sub.topic), &Validator{})
		}
		c.subscribe(sub.topic, sub)
	}

	return component
}

func (c *Consensus) subscribe(topic topics.Topic, sub Listener) {
	subscribers := c.subscribers[topic]
	if subscribers == nil {
		subscribers = []Listener{sub}
	} else {
		subscribers = append(subscribers, sub)
	}
	c.subscribers[topic] = subscribers
}

func (c *Consensus) reinstantiateComponents(fromScratch bool) {
	c.clearSubscribers()
	for i, factory := range c.factories {
		c.components[i].Finalize()
		component := factory.Instantiate()
		c.components[i] = c.initialize(component, fromScratch)
	}
}

func (c *Consensus) CollectRoundUpdate(m bytes.Buffer) error {
	r := RoundUpdate{}
	if err := DecodeRound(&m, &r); err != nil {
		return err
	}

	c.lock.Lock()
	c.reinstantiateComponents(false)
	c.lock.Unlock()
	return nil
}

func (c *Consensus) CollectEvent(m bytes.Buffer) error {
	topic, err := topics.Extract(&m)
	if err != nil {
		return err
	}

	// check header
	hdr := header.Header{}
	if err := header.Unmarshal(&m, &hdr); err != nil {
		return err
	}

	switch hdr.Compare(c.Round(), c.Step()) {
	case header.Before:
		// obsolete
		return nil
	case header.After:
		// TODO: queue the message
		return nil
	}

	c.lock.RLock()
	subscribers := c.subscribers[topic]
	c.lock.RUnlock()

	for _, sub := range subscribers {
		if err := sub.NotifyPayload(m, hdr); err != nil {
			//TODO: log this
		}
	}

	return nil
}

func (c *Consensus) requestStepUpdate() {
	c.IncrementStep()
	for _, component := range c.components {
		component.SetStep(c.Step())
	}
}

func (c *Consensus) clearSubscribers() {
	for k := range c.subscribers {
		delete(c.subscribers, k)
	}
}
