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

type roundStore struct {
	subscribers map[topics.Topic][]Listener
	components  []Component
	consensus   *Consensus
}

func newStore(c *Consensus) *roundStore {
	s := &roundStore{
		subscribers: make(map[topics.Topic][]Listener),
		components:  make([]Component, 0),
		consensus:   c,
	}

	return s
}

func (s *roundStore) RequestStepUpdate() {
	s.consensus.requestStepUpdate()
}

func (s *roundStore) addComponent(component Component, round RoundUpdate) []Subscriber {
	subs := component.Initialize(s, round)
	for _, sub := range subs {
		s.subscribe(sub.topic, sub)
	}
	s.components = append(s.components, component)
	return subs
}

func (s *roundStore) subscribe(topic topics.Topic, sub Listener) {
	subscribers := s.subscribers[topic]
	if subscribers == nil {
		subscribers = []Listener{sub}
	} else {
		subscribers = append(subscribers, sub)
	}
	s.subscribers[topic] = subscribers
}

func (s *roundStore) Dispatch(ev TopicEvent) {
	for _, sub := range s.subscribers[ev.Topic] {
		if err := sub.NotifyPayload(ev.Event); err != nil {
			//TODO: log this
		}
	}
}

func (s *roundStore) DispatchStepUpdate(step uint8) {
	for _, component := range s.components {
		component.SetStep(step)
	}
}

func (s *roundStore) DispatchFinalize() {
	for _, component := range s.components {
		component.Finalize()
	}
}

// Consensus encapsulates the information about the Round and the Step of the consensus. It also manages the roundStore,
// which aim is to centralize the state of the consensus Component while decoupling them from each other and the EventBus
type Consensus struct {
	*SyncState
	eventBus   *eventbus.EventBus
	rpcBus     *rpcbus.RPCBus
	keys       user.Keys
	factories  []ComponentFactory
	components []Component
	eventqueue *Queue

	lock  sync.RWMutex
	store *roundStore
}

// New creates a new Consensus
func New(eventBus *eventbus.EventBus, rpcBus *rpcbus.RPCBus, keys user.Keys, factories []ComponentFactory) *Consensus {
	c := &Consensus{
		SyncState:  NewState(),
		eventBus:   eventBus,
		rpcBus:     rpcBus,
		keys:       keys,
		factories:  factories,
		eventqueue: NewQueue(),
	}

	listener := eventbus.NewCallbackListener(c.CollectEvent)
	eventBus.SubscribeDefault(listener)

	// TODO: listen for initialization message and call recreateStore(roundUpdate, true)
	return c
}

func (c *Consensus) initialize(subs []Subscriber) {
	for _, sub := range subs {
		c.eventBus.AddDefaultTopic(sub.topic)
		c.eventBus.Register(sub.topic, NewRepublisher(c.eventBus, sub.topic), &Validator{})
	}
}

func (c *Consensus) recreateStore(roundUpdate RoundUpdate, fromScratch bool) {

	// reinstantiating the store prevents the need for locking
	store := newStore(c)

	for _, factory := range c.factories {
		component := factory.Instantiate()
		subs := store.addComponent(component, roundUpdate)
		if fromScratch {
			c.initialize(subs)
		}
	}

	c.swapStore(store)
}

func (c *Consensus) FinalizeRound() {
	c.store.DispatchFinalize()
	c.eventqueue.Clear(c.Round())
}

func (c *Consensus) CollectRoundUpdate(m bytes.Buffer) error {
	r := RoundUpdate{}
	if err := DecodeRound(&m, &r); err != nil {
		return err
	}
	c.FinalizeRound()

	c.recreateStore(r, false)
	c.Update(r.Round)
	return nil
}

// swapping stores is the only place that needs a lock as store is shared among the CollectRoundUpdate and
func (c *Consensus) swapStore(store *roundStore) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.store = store
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
		c.eventqueue.PutEvent(hdr.Round, hdr.Step, NewTopicEvent(topic, hdr, m))
		return nil
	}

	c.store.Dispatch(NewTopicEvent(topic, hdr, m))
	return nil
}

func (c *Consensus) requestStepUpdate() {
	c.IncrementStep()
	c.store.DispatchStepUpdate(c.Step())

	events := c.eventqueue.GetEvents(c.Round(), c.Step())
	for _, ev := range events {
		c.store.Dispatch(ev)
	}
}
