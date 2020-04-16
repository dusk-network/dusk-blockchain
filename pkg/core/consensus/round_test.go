package consensus

import (
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/stretchr/testify/assert"
)

var collectEventTable = []struct {
	eventRound        uint64
	eventStep         uint8
	receivedEventsLen int
	queuedEventsLen   int
}{
	{1, 0, 1, 0},
	{0, 0, 1, 0},
	{1, 1, 1, 1},
	{2, 0, 1, 1},
}

// Test that the coordinator redirects events correctly, according to their header
func TestCollectEvent(t *testing.T) {
	c, cmps := initCoordinatorTest(t, topics.Reduction)
	comp := cmps[0].(*mockComponent)

	for _, tt := range collectEventTable {
		ev := mockMessage(topics.Reduction, tt.eventRound, tt.eventStep)
		c.CollectEvent(ev)
		assert.Equal(t, tt.receivedEventsLen, len(comp.receivedEvents))
		assert.Equal(t, tt.queuedEventsLen, len(c.eventqueue.entries[tt.eventRound][tt.eventStep]))
	}
}

// Test that queued events are dispatched correctly on the appropriate state,
// when Play is called.
func TestQueuedDispatch(t *testing.T) {
	c, cmps := initCoordinatorTest(t, topics.Reduction)
	comp := cmps[0].(*mockComponent)

	// Send an event which should get queued
	ev := mockMessage(topics.Reduction, 1, 1)
	c.CollectEvent(ev)

	// Mock component should have no events saved
	assert.Equal(t, 0, len(comp.receivedEvents))
	// Queue should now hold one event on round 1, step 1
	assert.Equal(t, 1, len(c.eventqueue.entries[1][1]))

	// Pause, so that we can resume later
	c.Pause(comp.ID())
	// Forward to step 1. The queued event should be dispatched
	assert.Equal(t, uint8(1), c.Forward(comp.ID()))
	c.Play(comp.ID())
	assert.Equal(t, 1, len(comp.receivedEvents))
	assert.Equal(t, 0, len(c.eventqueue.entries[1][1]))

	// Send another event which should get queued for the next round
	ev = mockMessage(topics.Reduction, 2, 0)
	c.CollectEvent(ev)
	// Queue should now hold one event on round 2, step 0
	assert.Equal(t, 1, len(c.eventqueue.entries[2][0]))

	// Update to round 2. The queued event should be dispatched
	ru := MockRoundUpdate(2, nil, nil)
	msg := message.New(topics.RoundUpdate, ru)
	c.CollectRoundUpdate(msg)
	// Update our reference to `comp`, as it was swapped out.
	comp = c.store.components[0].(*mockComponent)
	c.Pause(comp.ID())
	c.Play(comp.ID())
	// Mock component should only hold one event, as it was re-instantiated
	// on the round update
	assert.Equal(t, 1, len(comp.receivedEvents))
	assert.Equal(t, 0, len(c.eventqueue.entries[2][0]))
}

// Test that events are withheld when a component is paused, and that streaming
// continues when resumed.
func TestPausePlay(t *testing.T) {
	c, cmps := initCoordinatorTest(t, topics.Reduction)
	comp := cmps[0].(*mockComponent)

	// Send an event with the correct state. It should be received
	ev := mockMessage(topics.Reduction, 1, 0)
	c.CollectEvent(ev)
	assert.Equal(t, 1, len(comp.receivedEvents))

	c.Pause(comp.id)

	// Send another event with the correct state. It should not be received
	ev = mockMessage(topics.Reduction, 1, 0)
	c.CollectEvent(ev)
	assert.Equal(t, 1, len(comp.receivedEvents))

	c.Play(comp.id)

	// Send one more event, which should be received.
	ev = mockMessage(topics.Reduction, 1, 0)
	c.CollectEvent(ev)
	assert.Equal(t, 2, len(comp.receivedEvents))
}

// Test that Agreement messages are filtered differently from other topics.
func TestEventFilter(t *testing.T) {
	c, comp := initCoordinatorTest(t, topics.Reduction, topics.Agreement)
	redComp := comp[0].(*mockComponent)
	agComp := comp[1].(*mockComponent)

	// Send a Reduction event with the correct state. It should be received
	ev := mockMessage(topics.Reduction, 1, 0)
	c.CollectEvent(ev)
	assert.Equal(t, 1, len(redComp.receivedEvents))

	// Send a Reduction event with future state. It should be queued
	ev = mockMessage(topics.Reduction, 1, 1)
	c.CollectEvent(ev)
	assert.Equal(t, 1, len(c.eventqueue.entries[1][1]))

	// Send an Agreement event with future state. It should be received
	ev = mockMessage(topics.Agreement, 1, 1)
	c.CollectEvent(ev)
	assert.Equal(t, 1, len(agComp.receivedEvents))
}

// Ensure an agreement event is queued on the correct state, and
// dispatched on the correct state.
func TestQueueDispatchAgreement(t *testing.T) {
	c, _ := initCoordinatorTest(t, topics.Agreement)

	// Send an Agreement event from a future round.
	// It should be queued
	ev := mockMessage(topics.Agreement, 2, 3)
	c.CollectEvent(ev)
	// Should be queued on round 2
	assert.Equal(t, 1, len(c.roundQueue.entries[2][3]))

	// Update the round to dispatch the event
	ru := MockRoundUpdate(2, nil, nil)
	msg := message.New(topics.RoundUpdate, ru)
	c.CollectRoundUpdate(msg)
	agComp := c.store.components[0].(*mockComponent)

	// Should receive the agreement event
	<-agComp.receivedEvents
}

// Initialize a coordinator with a single component.
func initCoordinatorTest(t *testing.T, tpcs ...topics.Topic) (*Coordinator, []Component) {
	bus := eventbus.New()
	keys, err := key.NewRandConsensusKeys()
	if err != nil {
		t.Fatal(err)
	}

	factories := make([]ComponentFactory, len(tpcs))
	for i, topic := range tpcs {
		factories[i] = &mockFactory{topic}
	}

	c := Start(bus, keys, factories...)
	ru := MockRoundUpdate(1, nil, nil)
	msg := message.New(topics.RoundUpdate, ru)

	// Collect the round update to initialize the state
	c.CollectRoundUpdate(msg)

	return c, c.store.components
}

func mockMessage(topic topics.Topic, round uint64, step uint8) message.Message {
	h := header.Header{
		Round:     round,
		Step:      step,
		BlockHash: make([]byte, 32),
		PubKeyBLS: make([]byte, 129),
	}
	return message.New(topic, h)
}

type mockFactory struct {
	topic topics.Topic
}

func (m *mockFactory) Instantiate() Component {
	return newMockComponent(m.topic)
}

// A dummy implementation of consensus.Component, used to check whether the Coordinator
// does it's job correctly.
type mockComponent struct {
	topic          topics.Topic
	receivedEvents chan InternalPacket
	id             uint32
}

func newMockComponent(topic topics.Topic) *mockComponent {
	return &mockComponent{
		topic:          topic,
		receivedEvents: make(chan InternalPacket, 100),
	}
}

func (m *mockComponent) Initialize(EventPlayer, Signer, RoundUpdate) []TopicListener {
	listener := TopicListener{
		Topic:    m.topic,
		Listener: NewSimpleListener(m.Collect, LowPriority, false),
	}
	m.id = listener.Listener.ID()

	return []TopicListener{listener}
}

func (m *mockComponent) ID() uint32 {
	return m.id
}

func (m *mockComponent) Collect(ev InternalPacket) error {
	m.receivedEvents <- ev
	return nil
}

func (m *mockComponent) Finalize() {}
