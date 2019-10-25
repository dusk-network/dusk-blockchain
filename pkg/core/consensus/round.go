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

var _ Stepper = (*Coordinator)(nil)
var _ Signer = (*Coordinator)(nil)

var lg = log.WithField("process", "coordinator")

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

func (s *roundStore) addComponent(component Component, round RoundUpdate) []TopicListener {
	subs := component.Initialize(s.coordinator, s.coordinator, s.coordinator, round)
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

func (s *roundStore) unsubscribe(id uint32) {
	for _, listeners := range s.subscribers {
		for i, listener := range listeners {
			if listener.ID() == id {
				listeners = append(
					listeners[:i],
					listeners[i+1:]...,
				)
				return
			}
		}
	}
}

func (s *roundStore) Dispatch(ev TopicEvent) {
	for _, sub := range s.subscribers[ev.Topic] {
		if err := sub.NotifyPayload(ev.Event); err != nil {
			//TODO: log this
		}
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

func (c *Coordinator) initialize(subs []TopicListener) {
	for _, sub := range subs {
		c.eventBus.AddDefaultTopic(sub.Topic)
		c.eventBus.Register(sub.Topic, sub.Preprocessors...)
	}
}

func (c *Coordinator) Subscribe(topic topics.Topic, listener Listener) {
	c.store.subscribe(topic, listener)
}

func (c *Coordinator) Unsubscribe(id uint32) {
	c.store.unsubscribe(id)
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

	switch hdr.Compare(c.Round(), c.Step()) {
	case header.Before:
		lg.WithField("topic", topic).Debugln("discarding obsolete event")
		return nil
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

func (c *Coordinator) RequestStepUpdate() {
	c.IncrementStep()

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
	buf, err := c.header(blockHash)
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
	if _, err := whole.ReadFrom(payload); err != nil {
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
	buf, err := c.header(hash)
	if err != nil {
		return err
	}

	if _, err := buf.ReadFrom(payload); err != nil {
		return err
	}

	c.eventBus.Publish(topic, &buf)
	return nil
}

// header reconstructs the header of a message from the blockHash
func (c *Coordinator) header(blockHash []byte) (bytes.Buffer, error) {
	buf := c.pubkeyBuf
	// ToBuffer is part of the SyncState struct
	stateBuf := c.ToBuffer()
	if _, err := buf.ReadFrom(&stateBuf); err != nil {
		return buf, err
	}

	if err := encoding.Write256(&buf, blockHash); err != nil {
		return buf, err
	}

	return buf, nil
}
