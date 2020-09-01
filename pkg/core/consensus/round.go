package consensus

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/capi"
	"github.com/dusk-network/dusk-blockchain/pkg/util/diagnostics"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-crypto/bls"
	log "github.com/sirupsen/logrus"
)

var _ EventPlayer = (*Coordinator)(nil)
var _ Signer = (*Coordinator)(nil)

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
	s := new(roundStore)
	s.Reset(c)
	return s
}

func (s *roundStore) Reset(c *Coordinator) {
	s.lock.Lock()
	defer s.lock.Unlock()

	// FIXME: should we pause the subscribers here ?
	//for _, listeners := range s.subscribers {
	//	for _, listener := range listeners {
	//		listener.Pause()
	//	}
	//}

	s.subscribers = make(map[topics.Topic][]Listener)
	s.components = make([]Component, 0)
	s.coordinator = c
}

func (s *roundStore) addComponent(component Component) {
	s.lock.Lock()
	s.components = append(s.components, component)
	s.lock.Unlock()
}

func (s *roundStore) hasComponent(id uint32) (bool, string) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	name := ""
	hasComponent := false
	i := sort.Search(len(s.components), func(i int) bool { return s.components[i].ID() >= id })
	hasComponent = i < len(s.components) && s.components[i].ID() == id
	if hasComponent {
		name = s.components[i].Name()
	}
	return hasComponent, name
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
func (s *roundStore) Dispatch(m message.Message) []error {
	s.lock.RLock()
	defer s.lock.RUnlock()
	var errorList []error
	subscribers := s.createSubscriberQueue(m.Category())
	lg.WithFields(log.Fields{
		"coordinator_round": s.coordinator.Round(),
		"coordinator_step":  s.coordinator.Step(),
		"recipients":        len(subscribers),
		"topic":             m.Category().String(),
	}).Traceln("notifying subscribers")
	for _, sub := range subscribers {
		ip := m.Payload().(InternalPacket)
		if err := sub.NotifyPayload(ip); err != nil {
			lg.WithFields(log.Fields{
				"coordinator_round": s.coordinator.Round(),
				"coordinator_step":  s.coordinator.Step(),
				"topic":             m.Category().String(),
				"round":             ip.State().Round,
				"step":              ip.State().Step,
				"id":                sub.ID(),
			}).WithError(err).Error("notifying subscriber failed, will panic")
			errorList = append(errorList, err)
		}
	}

	return errorList
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
	keys       key.Keys
	factories  []ComponentFactory
	eventqueue *Queue
	roundQueue *Queue

	pubkeyBuf bytes.Buffer

	lock     sync.RWMutex
	store    *roundStore
	unsynced bool

	stopped bool
}

// Start the coordinator by wiring the listener to the RoundUpdate
func Start(eventBus *eventbus.EventBus, keys key.Keys, factories ...ComponentFactory) *Coordinator {
	pkBuf := new(bytes.Buffer)

	if err := encoding.WriteVarBytes(pkBuf, keys.BLSPubKeyBytes); err != nil {
		log.Panic(err)
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
	collectEventListener := eventbus.NewCallbackListener(c.CollectEvent)
	collectRoundListener := eventbus.NewCallbackListener(c.CollectRoundUpdate)
	stopListener := eventbus.NewCallbackListener(c.StopConsensus)

	if config.Get().General.SafeCallbackListener {
		collectEventListener = eventbus.NewSafeCallbackListener(c.CollectEvent)
		collectRoundListener = eventbus.NewSafeCallbackListener(c.CollectRoundUpdate)
		stopListener = eventbus.NewSafeCallbackListener(c.StopConsensus)
	}

	c.eventBus.SubscribeDefault(collectEventListener)
	c.eventBus.Subscribe(topics.RoundUpdate, collectRoundListener)
	c.eventBus.Subscribe(topics.StopConsensus, stopListener)

	c.store = newStore(c)
	c.reinstantiateStore()
	return c
}

//StopConsensus stop the consensus for this round, finalizes the Round, instantiate a new Store
func (c *Coordinator) StopConsensus(m message.Message) error {
	if config.Get().API.Enabled {
		err := capi.StoreRoundInfo(c.round, c.step, "StopConsensus", "")
		if err != nil {
			lg.
				WithFields(log.Fields{
					"round": c.Round,
					"step":  c.Step,
				}).
				WithError(err).
				Error("could not save StoreRoundInfo on api db")
		}
	}
	log.
		WithField("round", c.Round()).
		Debug("StopConsensus")
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
func (c *Coordinator) CollectRoundUpdate(m message.Message) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	r := m.Payload().(RoundUpdate)

	if !c.stopped {
		c.stopConsensus()
	}
	c.onNewRound(r, c.unsynced)
	c.Update(r.Round)
	c.unsynced = false
	c.stopped = false
	go c.flushRoundQueue()

	// TODO: the Coordinator should not send events. Someone else should kickstart the
	// start consensus loop -> topics.Initialization ?

	log.
		WithField("round", r.Round).
		Debug("CollectRoundUpdate, Dispatch, topics.Generation")
	errList := c.store.Dispatch(message.New(topics.Generation, EmptyPacket()))
	if len(errList) > 0 {
		for _, err := range errList {
			log.
				WithError(err).
				WithField("round", r.Round).
				Error("failed to kickstart the consensus loop ? -> topics.Generation")
			//FIXME: shall this panic ? is this a extreme violation ?
		}
	}
	return nil
}

func (c *Coordinator) flushRoundQueue() {
	evs := c.roundQueue.Flush(c.Round())
	for _, ev := range evs {
		errList := c.store.Dispatch(ev)
		if len(errList) > 0 {
			for _, err := range errList {
				log.
					WithError(err).
					WithField("round", c.Round()).
					Error("failed to Dispatch flushRoundQueue")
			}
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
		"coordinator_step":  c.Step(),
		"coordinator_round": c.Round(),
		"message_round":     round,
	}).Debugln("received Finalize message")
	return nil
}

// Create a new roundStore and instantiate all Components.
func (c *Coordinator) reinstantiateStore() {
	c.store.Reset(c)
	for _, factory := range c.factories {
		component := factory.Instantiate()
		c.store.addComponent(component)
	}
}

// CollectEvent collects the consensus message and reroutes it to the proper
// component.
// It is the callback passed to the eventbus.Multicaster
func (c *Coordinator) CollectEvent(m message.Message) error {
	var msg InternalPacket
	switch p := m.Payload().(type) {
	case message.SafeBuffer: // TODO: we should actually panic here (panic??)
		_, _ = topics.Extract(&p)
		return fmt.Errorf("trying to feed the Coordinator a bytes.Buffer for message: %s", m.Category().String())
	case InternalPacket:
		msg = p
	default:
		return errors.New("trying to feed the Coordinator a screwed up message from the EventBus")
	}

	hdr := msg.State()
	lg.WithFields(log.Fields{
		"topic": m.Category().String(),
		"round": hdr.Round,
		"step":  hdr.Step,
	}).Traceln("collected event")

	var comparison header.Phase
	c.lock.RLock()
	defer c.lock.RUnlock()
	if m.Category() == topics.Agreement {
		comparison = hdr.CompareRound(c.Round())
	} else {
		comparison = hdr.CompareRoundAndStep(c.Round(), c.Step())
	}

	switch comparison {
	case header.Before:
		lg.
			WithFields(log.Fields{
				"topic":             m.Category().String(),
				"round":             hdr.Round,
				"step":              hdr.Step,
				"coordinator_round": c.Round(),
				"coordinator_step":  c.Step(),
			}).
			Debugln("discarding obsolete event")
		return nil
	case header.After:
		lg.
			WithFields(log.Fields{
				"topic":             m.Category().String(),
				"round":             hdr.Round,
				"step":              hdr.Step,
				"coordinator_round": c.Round(),
				"coordinator_step":  c.Step(),
			}).
			Debugln("storing future event")

		// If it is a future agreement event, we store it on the
		// `roundQueue`. This means that the event will be dispatched
		// as soon as the Coordinator reaches the round in the event
		// header.
		if m.Category() == topics.Agreement {
			c.roundQueue.PutEvent(hdr.Round, hdr.Step, m)
			return nil
		}

		// Otherwise, we just queue it according to the header round
		// and step.
		c.eventqueue.PutEvent(hdr.Round, hdr.Step, m)
		// store it here
		//TODO: should this be moved into eventqueue ?
		if config.Get().API.Enabled {
			err := capi.StoreEventQueue(hdr.Round, hdr.Step, m)
			if err != nil {
				lg.
					WithFields(log.Fields{
						"round": hdr.Round,
						"step":  hdr.Step,
					}).
					WithError(err).
					Error("could not save eventqueue on api db")
			}
		}

		return nil
	}

	errList := c.store.Dispatch(m)
	if len(errList) > 0 {
		for _, err := range errList {
			log.
				WithError(err).
				WithField("round", c.Round()).
				Error("failed to Dispatch CollectEvent")
		}
	}
	return nil
}

// FinalizeRound triggers the store to dispatch a finalize to the Components
// and clear the internal EventQueue
func (c *Coordinator) FinalizeRound() {
	if c.store != nil {
		c.store.DispatchFinalize()
		c.eventqueue.Clear(c.Round())
	}
}

// Forward complies to the EventPlayer interface. It increments the internal
// step count and returns it. It is used as a callback by the consensus
// components
func (c *Coordinator) Forward(id uint32) uint8 {
	hasComponent, name := c.store.hasComponent(id)
	if hasComponent {
		lg.
			WithField("step", c.Step()).
			WithField("name", name).
			WithField("round", c.Round()).
			WithField("id", id).
			Traceln("incrementing step")
		c.IncrementStep()
	}

	if config.Get().API.Enabled {
		_ = capi.StoreRoundInfo(c.round, c.step, "Forward", name)
	}
	return c.Step()
}

func (c *Coordinator) dispatchQueuedEvents() {
	events := c.eventqueue.GetEvents(c.Round(), c.Step())
	for _, ev := range events {
		errList := c.store.Dispatch(ev)
		if len(errList) > 0 {
			for _, err := range errList {
				log.
					WithError(err).
					WithField("round", c.Round()).
					Error("failed to Dispatch dispatchQueuedEvents")
			}
		}
	}
}

// Sign uses the blockhash (which is lost when decoupling the Header and the Payload) to recompose the Header and sign the Payload
// by adding it to the signature. Argument packet can be nil
// XXX: adjust the signature verification on reduction (and agreement)
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
// TODO: interface - marshaling should actually be done after the Gossip to
// respect the symmetry of the architecture
func (c *Coordinator) Gossip(msg message.Message, id uint32) error {
	hasComponent, _ := c.store.hasComponent(id)
	if !hasComponent {
		return fmt.Errorf("caller with ID %d is unregistered", id)
	}

	// message.Marshal takes care of prepending the topic, marshaling the
	// header, etc
	buf, err := message.Marshal(msg)
	if err != nil {
		return err
	}

	// TODO: interface - setting the payload to a buffer will go away as soon as the Marshaling
	// is performed where it is supposed to (i.e. after the Gossip)
	serialized := message.New(msg.Category(), buf)

	// gossip away
	errList := c.eventBus.Publish(topics.Gossip, serialized)
	diagnostics.LogPublishErrors("consensus/round.go, Coordinator, topics.Gossip", errList)
	return nil
}

// Compose complies with the consensus.Signer interface.
// It is a callback used by the consensus components to create the
// appropriate Header for the Consensus
func (c *Coordinator) Compose(pf PacketFactory) InternalPacket {
	return pf.Create(c.keys.BLSPubKeyBytes, c.Round(), c.Step())
}

// SendInternally publish a message for internal consumption (and therefore
// does not carry the topic, nor needs binary de-serialization)
func (c *Coordinator) SendInternally(topic topics.Topic, msg message.Message, id uint32) error {
	hasComponent, _ := c.store.hasComponent(id)
	if !hasComponent {
		return fmt.Errorf("caller with ID %d is unregistered", id)
	}
	errList := c.eventBus.Publish(topic, msg)
	diagnostics.LogPublishErrors("consensus/round.go, Coordinator, SendInternally", errList)
	return nil
}

/*
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
*/

// Pause event streaming for the listener with the specified ID.
func (c *Coordinator) Pause(id uint32) {
	lg.
		WithField("id", id).
		WithFields(log.Fields{
			"coordinator_round": c.Round(),
			"coordinator_step":  c.Step(),
		}).
		Traceln("pausing")
	c.store.pause(id)
}

// Play will resume event streaming for the listener with the specified ID.
func (c *Coordinator) Play(id uint32) {
	// Only dispatch events if a registered component asks to Resume
	if c.store.resume(id) {

		lg.
			WithField("id", id).
			WithFields(log.Fields{
				"coordinator_round": c.Round(),
				"coordinator_step":  c.Step(),
			}).
			Traceln("resumed")
		c.dispatchQueuedEvents()
	}
}
