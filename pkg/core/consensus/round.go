package consensus

import (
	"bytes"
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-crypto/bls"
	"github.com/dusk-network/dusk-wallet/key"
)

var _ Store = (*roundStore)(nil)

type roundStore struct {
	subscribers map[topics.Topic][]Listener
	components  []Component
	coordinator *Coordinator
}

func newStore(c *Coordinator) *roundStore {
	s := &roundStore{
		subscribers: make(map[topics.Topic][]Listener),
		components:  make([]Component, 0),
		coordinator: c,
	}

	return s
}

// WriteHeader writes the header to a payload before publishing the event
func (s *roundStore) WriteHeader(blockHash []byte, payload *bytes.Buffer) error {
	hBuf, err := s.coordinator.header(blockHash)
	if err != nil {
		return err
	}

	if _, err := hBuf.ReadFrom(payload); err != nil {
		return err
	}

	*payload = hBuf
	return nil
}

func (s *roundStore) RequestSignature(payload []byte) ([]byte, error) {
	return s.coordinator.sign(payload)
}

func (s *roundStore) RequestStepUpdate() {
	s.coordinator.requestStepUpdate()
}

func (s *roundStore) addComponent(component Component, round RoundUpdate) []Subscriber {
	subs := component.Initialize(s, round)
	for _, sub := range subs {
		s.subscribe(sub.Topic, sub)
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

// Coordinator encapsulates the information about the Round and the Step of the coordinator. It also manages the roundStore,
// which aim is to centralize the state of the coordinator Component while decoupling them from each other and the EventBus
type Coordinator struct {
	*SyncState
	eventBus   *eventbus.EventBus
	keys       key.ConsensusKeys
	factories  []ComponentFactory
	components []Component
	eventqueue *Queue

	pubkeyBuf bytes.Buffer

	lock     sync.RWMutex
	store    *roundStore
	unsynced bool
}

// Start the coordinator by wiring the listener to the RoundUpdate
func Start(eventBus *eventbus.EventBus, keys key.ConsensusKeys, factories ...ComponentFactory) *Coordinator {
	pkBuf := new(bytes.Buffer)

	if err := encoding.WriteVarBytes(pkBuf, keys.BLSPubKeyBytes); err != nil {
		panic(err)
	}

	c := &Coordinator{
		SyncState:  NewState(),
		eventBus:   eventBus,
		keys:       keys,
		factories:  factories,
		eventqueue: NewQueue(),
		pubkeyBuf:  *pkBuf,
		unsynced:   true,
	}

	// completing the initialization
	listener := eventbus.NewCallbackListener(c.CollectEvent)
	c.eventBus.SubscribeDefault(listener)

	l := eventbus.NewCallbackListener(c.CollectRoundUpdate)
	c.eventBus.Subscribe(topics.RoundUpdate, l)
	return c
}

func (c *Coordinator) initialize(subs []Subscriber) {
	for _, sub := range subs {
		c.eventBus.AddDefaultTopic(sub.Topic)
		// TODO: not all subs need a republisher and validator
		c.eventBus.Register(sub.Topic, NewRepublisher(c.eventBus, sub.Topic), &Validator{})
	}
}

func (c *Coordinator) recreateStore(roundUpdate RoundUpdate, fromScratch bool) {
	// reinstantiating the store prevents the need for locking
	store := newStore(c)

	for _, factory := range c.factories {
		component := factory.Instantiate()
		subs := store.addComponent(component, roundUpdate)
		if fromScratch && subs != nil {
			c.initialize(subs)
		}
	}

	c.swapStore(store)
}

func (c *Coordinator) FinalizeRound() {
	c.store.DispatchFinalize()
	c.eventqueue.Clear(c.Round())
}

func (c *Coordinator) CollectRoundUpdate(m bytes.Buffer) error {
	r := RoundUpdate{}
	if err := DecodeRound(&m, &r); err != nil {
		return err
	}
	// On the first round update, the store could still be nil.
	if c.store != nil {
		c.FinalizeRound()
	}

	c.recreateStore(r, c.unsynced)
	c.Update(r.Round)
	c.unsynced = false
	return nil
}

// swapping stores is the only place that needs a lock as store is shared among the CollectRoundUpdate and CollectEvent
func (c *Coordinator) swapStore(store *roundStore) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.store = store
}

func (c *Coordinator) CollectEvent(m bytes.Buffer) error {
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
		c.eventqueue.PutEvent(hdr.Round, hdr.Step, NewTopicEvent(topic, hdr, m))
		return nil
	}

	c.store.Dispatch(NewTopicEvent(topic, hdr, m))
	return nil
}

func (c *Coordinator) requestStepUpdate() {
	c.IncrementStep()
	c.store.DispatchStepUpdate(c.Step())

	events := c.eventqueue.GetEvents(c.Round(), c.Step())
	for _, ev := range events {
		c.store.Dispatch(ev)
	}
}

func (c *Coordinator) header(blockHash []byte) (bytes.Buffer, error) {
	buf := c.pubkeyBuf
	stateBuf := c.ToBuffer()
	if _, err := buf.ReadFrom(&stateBuf); err != nil {
		return buf, err
	}

	if err := encoding.Write256(&buf, blockHash); err != nil {
		return buf, err
	}

	return buf, nil
}

func (c *Coordinator) sign(payload []byte) ([]byte, error) {
	round, step := c.Round(), c.Step()
	buf := new(bytes.Buffer)
	if err := encoding.WriteUint64LE(buf, round); err != nil {
		return nil, err
	}

	if err := encoding.WriteUint8(buf, step); err != nil {
		return nil, err
	}

	if _, err := buf.Write(payload); err != nil {
		return nil, err
	}

	signedHash, err := bls.Sign(c.keys.BLSSecretKey, c.keys.BLSPubKey, payload)
	if err != nil {
		return nil, err
	}

	return signedHash.Compress(), nil
}
