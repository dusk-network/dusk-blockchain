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
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ed25519"
)

var _ EventPlayer = (*Coordinator)(nil)
var _ Signer = (*Coordinator)(nil)

var lg = log.WithField("process", "coordinator")

type roundStore struct {
	lock        sync.RWMutex
	subscribers map[topics.Topic][]Listener
	paused      map[topics.Topic][]Listener
	components  []Component
	coordinator *Coordinator
}

func newStore(c *Coordinator) *roundStore {
	s := &roundStore{
		subscribers: make(map[topics.Topic][]Listener),
		paused:      make(map[topics.Topic][]Listener),
		components:  make([]Component, 0),
		coordinator: c,
	}

	return s
}

func (s *roundStore) addComponent(component Component, round RoundUpdate) []TopicListener {
	subs := component.Initialize(s.coordinator, s.coordinator, round)
	for _, sub := range subs {
		s.subscribe(sub.Topic, sub)
		if sub.Paused {
			s.pause(sub.ID())
		}
	}
	s.lock.Lock()
	s.components = append(s.components, component)
	s.lock.Unlock()
	return subs
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
	s.lock.Lock()
	defer s.lock.Unlock()
	for topic, listeners := range s.subscribers {
		for i, listener := range listeners {
			if listener.ID() == id {
				listeners = append(
					listeners[:i],
					listeners[i+1:]...,
				)
				s.subscribers[topic] = listeners
				s.paused[topic] = append(s.paused[topic], listener)
				return
			}
		}
	}
}

func (s *roundStore) resume(id uint32) {
	s.lock.Lock()
	for topic, listeners := range s.paused {
		for i, listener := range listeners {
			if listener.ID() == id {
				s.paused[topic] = append(s.paused[topic][:i], s.paused[topic][i+1:]...)
				s.lock.Unlock()
				s.subscribe(topic, listener)
				return
			}
		}
	}

	s.lock.Unlock()
}

func (s *roundStore) Dispatch(ev TopicEvent) {
	s.lock.RLock()
	subscribers := s.createSubscriberQueue(ev.Topic)
	s.lock.RUnlock()
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
	subQueue := make([]Listener, 0, len(s.subscribers[topic]))
	for _, sub := range s.subscribers[topic] {
		if sub.Priority() == HighPriority {
			subQueue = append(subQueue, sub)
		}
	}

	for _, sub := range s.subscribers[topic] {
		if sub.Priority() == LowPriority {
			subQueue = append(subQueue, sub)
		}
	}

	return subQueue
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

func (c *Coordinator) initialize(subs []TopicListener) {
	for _, sub := range subs {
		c.eventBus.AddDefaultTopic(sub.Topic)
		c.eventBus.Register(sub.Topic, sub.Preprocessors...)
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

func (c *Coordinator) CollectRoundUpdate(m bytes.Buffer) error {
	lg.Debugln("received round update")
	r := RoundUpdate{}
	if err := DecodeRound(&m, &r); err != nil {
		return err
	}

	c.FinalizeRound()
	c.recreateStore(r, c.unsynced)
	c.Update(r.Round)
	c.unsynced = false
	return nil
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

	lg.WithFields(log.Fields{
		"topic": topic.String(),
		"round": hdr.Round,
		"step":  hdr.Step,
	}).Traceln("collected event")
	switch hdr.Compare(c.Round(), c.Step()) {
	case header.Before:
		// Unless it's an Agreement message, we should drop it if it's arriving late.
		if topic != topics.Agreement {
			lg.WithField("topic", topic).Debugln("discarding obsolete event")
			return nil
		}
	case header.After:
		lg.WithField("topic", topic).Debugln("storing future event")
		c.eventqueue.PutEvent(hdr.Round, hdr.Step, NewTopicEvent(topic, hdr, m))
		return nil
	}

	c.store.Dispatch(NewTopicEvent(topic, hdr, m))
	return nil
}

// swapping stores is the only place that needs a lock as store is shared among the CollectRoundUpdate and CollectEvent
func (c *Coordinator) swapStore(store *roundStore) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.store = store
}

func (c *Coordinator) FinalizeRound() {
	if c.store != nil {
		c.store.DispatchFinalize()
		c.eventqueue.Clear(c.Round())
	}
}

func (c *Coordinator) Play() uint8 {
	c.IncrementStep()
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
func (c *Coordinator) Sign(hash, packet []byte) ([]byte, error) {
	preimage := new(bytes.Buffer)
	h := header.Header{
		Round:     c.Round(),
		Step:      c.Step(),
		BlockHash: hash,
		PubKeyBLS: c.keys.BLSPubKeyBytes,
	}

	if err := header.MarshalSignableVote(preimage, h, packet); err != nil {
		return nil, err
	}

	signedHash, err := bls.Sign(c.keys.BLSSecretKey, c.keys.BLSPubKey, preimage.Bytes())
	if err != nil {
		return nil, err
	}

	return signedHash.Compress(), nil
}

// SendAuthenticated sign the payload with Ed25519 and publishes it to the Gossip topic
func (c *Coordinator) SendAuthenticated(topic topics.Topic, blockHash []byte, payload *bytes.Buffer) error {
	buf, err := header.Compose(c.pubkeyBuf, c.ToBuffer(), blockHash)
	if err != nil {
		return err
	}

	if _, err := buf.ReadFrom(payload); err != nil {
		return err
	}

	edSigned := ed25519.Sign(*c.keys.EdSecretKey, buf.Bytes())

	// messages start from the signature
	whole := new(bytes.Buffer)
	if err := encoding.Write512(whole, edSigned); err != nil {
		return err
	}

	// adding Ed public key
	if err := encoding.Write256(whole, c.keys.EdPubKeyBytes); err != nil {
		return err
	}

	// adding marshalled header+payload
	if _, err := whole.ReadFrom(&buf); err != nil {
		return err
	}

	// prepending topic
	if err := topics.Prepend(whole, topic); err != nil {
		return err
	}

	// gossip away
	c.eventBus.Publish(topics.Gossip, whole)
	return nil
}

// SendWithHeader prepends a header to the given payload, and publishes it on the
// desired topic.
func (c *Coordinator) SendWithHeader(topic topics.Topic, hash []byte, payload *bytes.Buffer) error {
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

func (c *Coordinator) Pause(id uint32) {
	c.store.pause(id)
}

func (c *Coordinator) Resume(id uint32) {
	c.store.resume(id)
	c.dispatchQueuedEvents()
}
