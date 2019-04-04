package notary

import (
	"bytes"
	"encoding/binary"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

func TestSigSetNotary(t *testing.T) {

	bus, _, _ := initNotary(1)
	roundChan := make(chan *bytes.Buffer)
	bus.Subscribe(msg.RoundUpdateTopic, roundChan)
	bus.Publish(string(topics.SigSetAgreement), bytes.NewBuffer([]byte("test")))

	select {
	case result := <-roundChan:
		round := binary.LittleEndian.Uint64(result.Bytes())
		assert.Equal(t, uint64(2), round)
	case <-time.After(100 * time.Millisecond):
		assert.FailNow(t, "SigSetNotary should have returned (or you can try to relax the timeout)")
	}
}

func TestFutureRounds(t *testing.T) {
	bus, collector, _ := initNotary(1)
	//the Unmarshaller unmarshals messages for a future round
	collector.Unmarshaller = newMockSEUnmarshaller([]byte("whatever"), 2, 1)

	roundChan := make(chan *bytes.Buffer)
	bus.Subscribe(msg.RoundUpdateTopic, roundChan)
	bus.Publish(string(topics.SigSetAgreement), bytes.NewBuffer([]byte("test")))

	select {
	case <-roundChan:
		assert.FailNow(t, "No round update should have been propagated since the event refers to a future round")
	case <-time.After(100 * time.Millisecond):
		// success
		assert.Equal(t, 1, collector.futureRounds.Len())
	}
}

func TestProcessFutureRounds(t *testing.T) {
	bus, collector, _ := initNotary(2)
	//the Unmarshaller unmarshals messages for a future round
	collector.Unmarshaller = newMockSEUnmarshaller([]byte("whatever"), 2, 1)

	roundChan := make(chan *bytes.Buffer)
	bus.Subscribe(msg.RoundUpdateTopic, roundChan)
	// accumulating messages for future rounds to be processed
	bus.Publish(string(topics.SigSetAgreement), bytes.NewBuffer([]byte("test")))
	bus.Publish(string(topics.SigSetAgreement), bytes.NewBuffer([]byte("test")))
	<-time.After(50 * time.Millisecond)

	// triggering a round update
	//setting the mock to unmarshal messages for current round
	collector.Unmarshaller.(*mockSEUnmarshaller).yield([]byte("whatever"), 1, 1)
	bus.Publish(string(topics.SigSetAgreement), bytes.NewBuffer([]byte("test")))
	bus.Publish(string(topics.SigSetAgreement), bytes.NewBuffer([]byte("test")))

	for i := 0; i < 2; i++ {
		select {
		case roundUpdate := <-roundChan:
			round := binary.LittleEndian.Uint64(roundUpdate.Bytes())
			switch i {
			case 0:
				assert.Equal(t, uint64(2), round)
				return
			case 1:
				assert.Equal(t, uint64(3), round)
				return
			}
		case <-time.After(100 * time.Millisecond):
			// success
			assert.FailNow(t, "Time out in receiving 2 round updates")
		}
	}
}

func initNotary(quorum int) (*wire.EventBus, *sigSetCollector, committee.Committee) {
	bus := wire.New()
	committee := mockCommittee(quorum, true, nil)
	notary := LaunchSignatureSetNotary(bus, committee, uint64(1))
	notary.sigSetCollector.Unmarshaller = newMockSEUnmarshaller([]byte("mock"), 1, 1)
	notary.sigSetCollector.futureRounds = &mockMap{newFutureMap(), &sync.Mutex{}}
	return bus, notary.sigSetCollector, committee
}

type mockMap struct {
	*futureMap
	lock *sync.Mutex
}

func (f *mockMap) Set(k uint64, v *SigSetEvent) int {
	f.lock.Lock()
	defer f.lock.Unlock()
	return f.futureMap.Set(k, v)
}

func (f *mockMap) Get(k uint64) []*SigSetEvent {
	f.lock.Lock()
	defer f.lock.Unlock()
	return f.entries[k]
}

func (f *mockMap) Len() int {
	f.lock.Lock()
	defer f.lock.Unlock()
	return len(f.entries)
}

type mockSEUnmarshaller struct {
	event *SigSetEvent
	err   error
	lock  *sync.Mutex
}

func (m *mockSEUnmarshaller) Unmarshal(b *bytes.Buffer, e wire.Event) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.err != nil {
		return m.err
	}

	blsPub, _ := crypto.RandEntropy(32)
	ev := e.(*SigSetEvent)
	ev.Step = m.event.Step
	ev.Round = m.event.Round
	ev.BlockHash = m.event.BlockHash
	ev.PubKeyBLS = blsPub
	return nil
}

func (m *mockSEUnmarshaller) yield(blockHash []byte, round uint64, step uint8) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.event.BlockHash = blockHash
	m.event.Round = round
	m.event.Step = step
}

// Marshal is not used by the mock unmarshaller.
func (m *mockSEUnmarshaller) Marshal(b *bytes.Buffer, e wire.Event) error {
	return nil
}

func newMockSEUnmarshaller(blockHash []byte, round uint64, step uint8) wire.EventUnMarshaller {
	ev := NewSigSetEvent()
	ev.BlockHash = blockHash
	ev.Round = round
	ev.Step = step

	return &mockSEUnmarshaller{
		event: ev,
		err:   nil,
		lock:  &sync.Mutex{},
	}
}
