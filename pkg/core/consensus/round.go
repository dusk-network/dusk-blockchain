package consensus

import (
	"bytes"
	"fmt"
	"sort"
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/marshalling"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/peermsg"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-crypto/bls"
	"github.com/dusk-network/dusk-wallet/block"
	"github.com/dusk-network/dusk-wallet/key"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ed25519"
)

var _ EventPlayer = (*Coordinator)(nil)
var _ Signer = (*Coordinator)(nil)
var emptyHash [32]byte
var emptyPayload = new(bytes.Buffer)

var lg = log.WithField("process", "coordinator")

// RoundState contains all the information for a component to be able
// to do it's job in a given round
type RoundState struct {
	RoundUpdate
	Seed        []byte
	BlockHash   []byte
	Certificate block.Certificate
}

// roundStore is the central registry for all consensus components and listeners.
// It is used for message dispatching and controlling the stream of events.
type roundStore struct {
	lock              sync.RWMutex
	subscribers       map[topics.Topic][]Listener
	components        []Component
	coordinator       *Coordinator
	certificate       *block.Certificate
	intermediateBlock *block.Block
}

func newStore(c *Coordinator, cert *block.Certificate, intermediate *block.Block) *roundStore {
	s := &roundStore{
		subscribers:       make(map[topics.Topic][]Listener),
		components:        make([]Component, 0),
		coordinator:       c,
		certificate:       cert,
		intermediateBlock: intermediate,
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

func (s *roundStore) initializeComponents(round RoundState) []TopicListener {
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
	rpcBus     *rpcbus.RPCBus
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
func Start(eventBus *eventbus.EventBus, rpcBus *rpcbus.RPCBus, keys key.ConsensusKeys, factories ...ComponentFactory) *Coordinator {
	pkBuf := new(bytes.Buffer)

	if err := encoding.WriteVarBytes(pkBuf, keys.BLSPubKeyBytes); err != nil {
		panic(err)
	}

	c := &Coordinator{
		SyncState:  NewState(),
		eventBus:   eventBus,
		rpcBus:     rpcBus,
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

	finalizeListener := eventbus.NewCallbackListener(c.CollectFinalize)
	c.eventBus.Subscribe(topics.Finalize, finalizeListener)

	c.reinstantiateStore(nil, nil)
	return c
}

func (c *Coordinator) onNewRound(roundState RoundState, fromScratch bool) {
	subs := c.store.initializeComponents(roundState)
	if fromScratch && subs != nil {
		for _, sub := range subs {
			c.eventBus.AddDefaultTopic(sub.Topic)
			c.eventBus.Register(sub.Topic, sub.Preprocessors...)
		}
	}
}

// CollectRoundUpdate is triggered when the Chain propagates a new round update.
// If the Finalize message was not seen earlier, it will finalize all components,
// reinstantiate a new store, and swap it with the current one. The consensus
// components are then initialized, and the state will be updated to the new round.
func (c *Coordinator) CollectRoundUpdate(m bytes.Buffer) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	r := RoundUpdate{}
	if err := DecodeRound(&m, &r); err != nil {
		return err
	}
	lg.WithField("round", r.Round).Debugln("received round update")

	// If the round in the round update tells us that we're behind,
	// we go into Dozer mode.
	if r.Round > c.Round()+1 {
		c.doze(r)
		return nil
	}

	// Check that we have finalized and created a certificate first
	// Otherwise, we also go into dozer mode.
	c.store.lock.RLock()
	lenSubs := len(c.store.subscribers)
	c.store.lock.RUnlock()
	if lenSubs > 0 {
		c.doze(r)
		return nil
	}
	c.Update(r.Round)
	rs := RoundState{
		RoundUpdate: r,
	}

	if c.store.intermediateBlock != nil {
		rs.Seed = c.store.intermediateBlock.Header.Seed
		rs.BlockHash = c.store.intermediateBlock.Header.Hash
	}

	if c.store.certificate != nil {
		rs.Certificate = *c.store.certificate
	}

	c.onNewRound(rs, c.unsynced)
	c.unsynced = false

	// TODO: the Coordinator should not send events. someone else should kickstart the
	// consensus loop -- Generation component?
	c.store.Dispatch(TopicEvent{
		Topic: topics.Generation,
		Event: Event{},
	})
	return nil
}

// Enter dozer mode. This means that we are one round behind the
// consensus execution trace. We will finalize the current round,
// instantiate an empty Store, and query for Agreement messages
// from our peers, so that we can catch up.
func (c *Coordinator) doze(r RoundUpdate) {
	lg.Traceln("finalizing consensus, going into dozer mode")
	// Clear all of the consensus state, so that we can ask for
	// Agreement messages without possibly accidentally triggering
	// finalization
	c.FinalizeRound()
	c.reinstantiateStore(nil, nil)
	rs := RoundState{r, nil, nil, *block.EmptyCertificate()}
	c.onNewRound(rs, c.unsynced)
	// We can't skip ahead to the latest round as we are missing the
	// agreement messages for the previous one, so we will go to the
	// one before it.
	c.Update(r.Round - 1)
	c.unsynced = false
	// Query for Agreement messages belonging to this round
	c.requestAgreementMessages(c.Round())
}

func (c *Coordinator) requestAgreementMessages(round uint64) {
	msg := &peermsg.GetAgreements{round}
	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		panic(err)
	}

	if err := topics.Prepend(buf, topics.GetAgreements); err != nil {
		panic(err)
	}

	c.eventBus.Publish(topics.Gossip, buf)
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

	winningBlockHash := make([]byte, 32)
	if err := encoding.Read256(&m, winningBlockHash); err != nil {
		return err
	}

	cert := &block.Certificate{}
	if err := marshalling.UnmarshalCertificate(&m, cert); err != nil {
		return err
	}
	lg.WithFields(log.Fields{
		"coordinator round": c.Round(),
		"message round":     round,
	}).Debugln("received Finalize message")

	if round < c.Round() {
		return nil
	}

	if round > c.Round() {
		panic("not supposed to get a Finalize message for a future round")
	}

	// If we have one, finalize current intermediate block, and send to chain
	if c.store.intermediateBlock != nil {
		if err := c.finalizeIntermediateBlock(cert); err != nil {
			return err
		}
	}

	// Fetch the winning candidate
	winningCandidate := block.NewBlock()
	req := rpcbus.Request{
		Params:   *bytes.NewBuffer(winningBlockHash),
		RespChan: make(chan rpcbus.Response, 1),
	}
	winningCandidateBuf, err := c.rpcBus.Call(rpcbus.GetCandidate, req, 0)
	if err != nil {
		// If we don't have it, request it from peers
		winningCandidateBuf = c.requestIntermediateBlock(winningBlockHash)
	}

	if err := marshalling.UnmarshalBlock(&winningCandidateBuf, winningCandidate); err != nil {
		return err
	}

	c.finalizeConsensus(cert, winningCandidate)
	return nil
}

func (c *Coordinator) requestIntermediateBlock(blockHash []byte) bytes.Buffer {
	return bytes.Buffer{}
}

func (c *Coordinator) finalizeConsensus(cert *block.Certificate, blk *block.Block) {
	lg.Traceln("finalizing consensus")
	// reinstantiating the store prevents the need for locking
	c.FinalizeRound()
	c.reinstantiateStore(cert, blk)
}

func (c *Coordinator) finalizeIntermediateBlock(cert *block.Certificate) error {
	candidate := c.store.intermediateBlock
	candidate.Header.Certificate = cert
	buf := new(bytes.Buffer)
	if err := marshalling.MarshalBlock(buf, candidate); err != nil {
		return err
	}

	c.eventBus.Publish(topics.Block, buf)
	return nil
}

// Create a new roundStore and instantiate all Components.
func (c *Coordinator) reinstantiateStore(cert *block.Certificate, intermediate *block.Block) {
	store := newStore(c, cert, intermediate)
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

// SendAuthenticated sign the payload with Ed25519 and publishes it to the Gossip topic
func (c *Coordinator) SendAuthenticated(topic topics.Topic, hdr header.Header, payload *bytes.Buffer, id uint32) error {
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
	if _, err := whole.ReadFrom(buf); err != nil {
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
func (c *Coordinator) SendWithHeader(topic topics.Topic, hash []byte, payload *bytes.Buffer, id uint32) error {
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
