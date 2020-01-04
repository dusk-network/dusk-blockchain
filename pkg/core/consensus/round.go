package consensus

import (
	"bytes"
	"fmt"
	"sort"
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-crypto/bls"
	"github.com/dusk-network/dusk-wallet/key"
	log "github.com/sirupsen/logrus"
)

var _ EventPlayer = (*Coordinator)(nil)
var _ Signer = (*Coordinator)(nil)
var emptyHash [32]byte
var emptyPayload = new(bytes.Buffer)

var lg = log.WithField("process", "coordinator")

// roundStore is the central registry for all consensus components and listeners.
// It is used for message dispatching and controlling the stream of events.
type roundStore struct {
	lock        sync.RWMutex
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

func (s *roundStore) addComponent(component Component) {
	s.lock.Lock()
	s.components = append(s.components, component)
	s.lock.Unlock()
}

func (s *roundStore) hasComponent(id uint32) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	i := sort.Search(len(s.components), func(i int) bool { return s.components[i].ID() >= id })
	return i < len(s.components) && s.components[i].ID() == id
}

func (s *roundStore) initializeComponents(round RoundUpdate) []TopicListener {
	allSubs := make([]TopicListener, 0, len(s.components)*2)
	for _, component := range s.components {
		subs := component.Initialize(s.coordinator, s.coordinator, round)
		for _, sub := range subs {
			s.subscribe(sub.Topic, sub.Listener)
		}

		allSubs = append(allSubs, subs...)
	}
	s.lock.Lock()
	sort.Slice(s.components, func(i, j int) bool { return s.components[i].ID() < s.components[j].ID() })
	s.lock.Unlock()

	return allSubs
}

func (s *roundStore) subscribe(topic topics.Topic, sub Listener) {
	s.lock.Lock()
	defer s.lock.Unlock()
	subscribers := s.subscribers[topic]
	if subscribers == nil {
		subscribers = []Listener{sub}
	} else {
		subscribers = append(subscribers, sub)
	}
	s.subscribers[topic] = subscribers
}

func (s *roundStore) pause(id uint32) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	for _, listeners := range s.subscribers {
		for _, listener := range listeners {
			if listener.ID() == id {
				listener.Pause()
				return
			}
		}
	}
}

func (s *roundStore) resume(id uint32) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	for _, listeners := range s.subscribers {
		for _, listener := range listeners {
			if listener.ID() == id {
				listener.Resume()
				return true
			}
		}
	}

	return false
}

// Dispatch an event to listeners for the designated Topic.
func (s *roundStore) Dispatch(ev TopicEvent) {
	subscribers := s.createSubscriberQueue(ev.Topic)
	lg.WithFields(log.Fields{
		"recipients": len(subscribers),
		"topic":      ev.Topic,
	}).Traceln("notifying subscribers")
	for _, sub := range subscribers {
		if err := sub.NotifyPayload(ev.Event); err != nil {
			lg.WithFields(log.Fields{
				"topic": ev.Topic.String(),
				"id":    sub.ID(),
			}).WithError(err).Warnln("notifying subscriber failed")
		}
	}
}

// order subscribers by priority for event dispatch
// TODO: it makes more sense to do this once per round
func (s *roundStore) createSubscriberQueue(topic topics.Topic) []Listener {
	s.lock.RLock()
	defer s.lock.RUnlock()
	subQueue := make([]Listener, 0, len(s.subscribers[topic]))
	for _, sub := range s.subscribers[topic] {
		if sub.Priority() == HighPriority && !sub.Paused() {
			subQueue = append(subQueue, sub)
		}
	}

	for _, sub := range s.subscribers[topic] {
		if sub.Priority() == LowPriority && !sub.Paused() {
			subQueue = append(subQueue, sub)
		}
	}

	return subQueue
}

// DispatchFinalize will finalize all components on the roundStore. The store is
// considered obsolete after calling this method.
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
	roundQueue *Queue

	pubkeyBuf bytes.Buffer

	lock     sync.RWMutex
	store    *roundStore
	unsynced bool

	stopped bool
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
		roundQueue: NewQueue(),
		pubkeyBuf:  *pkBuf,
		unsynced:   true,
		stopped:    true,
	}

	// completing the initialization
	listener := eventbus.NewCallbackListener(c.CollectEvent)
	c.eventBus.SubscribeDefault(listener)

	l := eventbus.NewCallbackListener(c.CollectRoundUpdate)
	c.eventBus.Subscribe(topics.RoundUpdate, l)

	stopListener := eventbus.NewCallbackListener(c.StopConsensus)
	c.eventBus.Subscribe(topics.StopConsensus, stopListener)

	c.reinstantiateStore()
	return c
}

func (c *Coordinator) StopConsensus(bytes.Buffer) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	if !c.stopped {
		c.stopConsensus()
		c.stopped = true
	}

	return nil
}

func (c *Coordinator) stopConsensus() {
	c.FinalizeRound()
	c.reinstantiateStore()
}

func (c *Coordinator) onNewRound(roundUpdate RoundUpdate, fromScratch bool) {
	subs := c.store.initializeComponents(roundUpdate)
	if fromScratch && subs != nil {
		for _, sub := range subs {
			c.eventBus.AddDefaultTopic(sub.Topic)
		}
	}
}

// CollectRoundUpdate is triggered when the Chain propagates a new round update.
// The consensus components are swapped out, initialized, and the
// state will be updated to the new round.
func (c *Coordinator) CollectRoundUpdate(m bytes.Buffer) error {
	lg.Debugln("received round update")
	c.lock.Lock()
	defer c.lock.Unlock()
	r := RoundUpdate{}
	if err := DecodeRound(&m, &r); err != nil {
		return err
	}

	if !c.stopped {
		c.stopConsensus()
	}
	c.onNewRound(r, c.unsynced)
	c.Update(r.Round)
	c.unsynced = false
	c.stopped = false
	go c.flushRoundQueue()

	// TODO: the Coordinator should not send events. someone else should kickstart the
	// consensus loop
	c.store.Dispatch(TopicEvent{
		Topic: topics.Generation,
		Event: Event{},
	})
	return nil
}

func (c *Coordinator) flushRoundQueue() {
	evs := c.roundQueue.Flush(c.Round())
	if evs != nil {
		for _, ev := range evs {
			c.store.Dispatch(ev)
		}
	}
}

// CollectFinalize is triggered when the Agreement reaches quorum, and pre-emptively
// finalizes all consensus components, as they are no longer needed after this point.
func (c *Coordinator) CollectFinalize(m bytes.Buffer) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	// Ensure that we should take this message seriously
	var round uint64
	if err := encoding.ReadUint64LE(&m, &round); err != nil {
		return err
	}
	lg.WithFields(log.Fields{
		"coordinator round": c.Round(),
		"message round":     round,
	}).Debugln("received Finalize message")

	return nil
}

// Create a new roundStore and instantiate all Components.
func (c *Coordinator) reinstantiateStore() {
	store := newStore(c)
	for _, factory := range c.factories {
		component := factory.Instantiate()
		store.addComponent(component)
	}

	c.swapStore(store)
}

func (c *Coordinator) CollectEvent(m bytes.Buffer) error {
	// NOTE: RUnlock is not deferred here, for performance reasons.
	// https://medium.com/i0exception/runtime-overhead-of-using-defer-in-go-7140d5c40e32
	// TODO: once go 1.14 is out, re-examine the overhead of using `defer`.
	c.lock.RLock()
	topic, err := topics.Extract(&m)
	if err != nil {
		c.lock.RUnlock()
		return err
	}

	// check header
	hdr := header.Header{}
	if err := header.Unmarshal(&m, &hdr); err != nil {
		c.lock.RUnlock()
		return err
	}

	lg.WithFields(log.Fields{
		"topic": topic.String(),
		"round": hdr.Round,
		"step":  hdr.Step,
	}).Traceln("collected event")

	var comparison header.Phase
	if topic == topics.Agreement {
		comparison = hdr.CompareRound(c.Round())
	} else {
		comparison = hdr.CompareRoundAndStep(c.Round(), c.Step())
	}

	switch comparison {
	case header.Before:
		lg.WithField("topic", topic).Debugln("discarding obsolete event")
		c.lock.RUnlock()
		return nil
	case header.After:
		lg.WithField("topic", topic).Debugln("storing future event")

		// If it is a future agreement event, we store it on the
		// `roundQueue`. This means that the event will be dispatched
		// as soon as the Coordinator reaches the round in the event
		// header.
		if topic == topics.Agreement {
			c.roundQueue.PutEvent(hdr.Round, hdr.Step, NewTopicEvent(topic, hdr, m))
			c.lock.RUnlock()
			return nil
		}

		// Otherwise, we just queue it according to the header round
		// and step.
		c.eventqueue.PutEvent(hdr.Round, hdr.Step, NewTopicEvent(topic, hdr, m))
		c.lock.RUnlock()
		return nil
	}

	c.store.Dispatch(NewTopicEvent(topic, hdr, m))
	c.lock.RUnlock()
	return nil
}

// swapping stores is the only place that needs a lock as store is shared among the CollectRoundUpdate and CollectEvent
func (c *Coordinator) swapStore(store *roundStore) {
	c.store = store
}

func (c *Coordinator) FinalizeRound() {
	if c.store != nil {
		c.store.DispatchFinalize()
		c.eventqueue.Clear(c.Round())
	}
}

func (c *Coordinator) Forward(id uint32) uint8 {
	if c.store.hasComponent(id) {
		lg.WithField("id", id).Traceln("incrementing step")
		c.IncrementStep()
	}
	return c.Step()
}

func (c *Coordinator) dispatchQueuedEvents() {
	events := c.eventqueue.GetEvents(c.Round(), c.Step())
	for _, ev := range events {
		c.store.Dispatch(ev)
	}
}

// XXX: adjust the signature verification on reduction (and agreement)
// Sign uses the blockhash (which is lost when decoupling the Header and the Payload) to recompose the Header and sign the Payload
// by adding it to the signature. Argument packet can be nil
func (c *Coordinator) Sign(h header.Header) ([]byte, error) {
	preimage := new(bytes.Buffer)
	if err := header.MarshalSignableVote(preimage, h); err != nil {
		return nil, err
	}

	signedHash, err := bls.Sign(c.keys.BLSSecretKey, c.keys.BLSPubKey, preimage.Bytes())
	if err != nil {
		return nil, err
	}

	return signedHash.Compress(), nil
}

// Gossip concatenates the topic, the header and the payload,
// and gossips it to the rest of the network.
func (c *Coordinator) Gossip(topic topics.Topic, hdr header.Header, payload *bytes.Buffer, id uint32) error {
	if !c.store.hasComponent(id) {
		return fmt.Errorf("caller with ID %d is unregistered", id)
	}

	buf := new(bytes.Buffer)
	if err := header.Marshal(buf, hdr); err != nil {
		return err
	}

	if _, err := buf.ReadFrom(payload); err != nil {
		return err
	}

	// prepending topic
	if err := topics.Prepend(buf, topic); err != nil {
		return err
	}

	// gossip away
	c.eventBus.Publish(topics.Gossip, buf)
	return nil
}

// SendInternally prepends a header to the given payload, and publishes
// it on the desired topic.
func (c *Coordinator) SendInternally(topic topics.Topic, hash []byte, payload *bytes.Buffer, id uint32) error {
	if !c.store.hasComponent(id) {
		return fmt.Errorf("caller with ID %d is unregistered", id)
	}

	lg.WithField("topic", topic.String()).Debugln("sending event internally")
	buf, err := header.Compose(c.pubkeyBuf, c.ToBuffer(), hash)
	if err != nil {
		lg.Traceln("could not compose header")
		return err
	}

	if _, err := buf.ReadFrom(payload); err != nil {
		lg.Traceln("could not extract payload")
		return err
	}

	c.eventBus.Publish(topic, &buf)
	return nil
}

// Pause event streaming for the listener with the specified ID.
func (c *Coordinator) Pause(id uint32) {
	lg.WithField("id", id).Traceln("pausing")
	c.store.pause(id)
}

// Play will resume event streaming for the listener with the specified ID.
func (c *Coordinator) Play(id uint32) {
	// Only dispatch events if a registered component asks to Resume
	if c.store.resume(id) {
		lg.WithField("id", id).Traceln("resumed")
		c.dispatchQueuedEvents()
	}
}
