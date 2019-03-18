package notary

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

func TestSigSetNotary(t *testing.T) {

	bus, _, _ := initNotary()
	roundChan := make(chan *bytes.Buffer)
	bus.Subscribe(msg.RoundUpdateTopic, roundChan)
	bus.Publish(msg.SigSetAgreementTopic, bytes.NewBuffer([]byte("test")))

	select {
	case result := <-roundChan:
		round := binary.LittleEndian.Uint64(result.Bytes())
		assert.Equal(t, uint64(2), round)
	case <-time.After(100 * time.Millisecond):
		assert.FailNow(t, "SigSetNotary should have returned (or you can try to relax the timeout)")
	}
}

func TestFutureRounds(t *testing.T) {
	bus, collector, _ := initNotary()
	//the Unmarshaller unmarshals messages for a future round
	collector.Unmarshaller = newMockSEUnmarshaller([]byte("whatever"), 2, 1)

	roundChan := make(chan *bytes.Buffer)
	bus.Subscribe(msg.RoundUpdateTopic, roundChan)
	bus.Publish(msg.SigSetAgreementTopic, bytes.NewBuffer([]byte("test")))

	select {
	case <-roundChan:
		assert.FailNow(t, "No round update should have been propagated since the event refers to a future round")
	case <-time.After(100 * time.Millisecond):
		// success
		assert.Equal(t, 1, len(collector.futureRounds))
	}
}

func initNotary() (*wire.EventBus, *SigSetCollector, user.Committee) {
	bus := wire.New()
	committee := mockCommittee(1, true, nil)
	notary := NewSigSetNotary(bus, nil, committee, uint64(1))

	notary.sigSetCollector.Unmarshaller = newMockSEUnmarshaller([]byte("mock"), 1, 1)
	go notary.Listen()
	return bus, notary.sigSetCollector, committee
}

type mockSEUnmarshaller struct {
	event *SigSetEvent
	err   error
}

func (m *mockSEUnmarshaller) Unmarshal(b *bytes.Buffer, e Event) error {
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

func newMockSEUnmarshaller(blockHash []byte, round uint64, step uint8) EventUnmarshaller {
	ev := &SigSetEvent{
		committeeEvent: &committeeEvent{},
	}
	ev.BlockHash = blockHash
	ev.Round = round
	ev.Step = step

	return &mockSEUnmarshaller{
		event: ev,
		err:   nil,
	}
}
