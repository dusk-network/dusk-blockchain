package reduction

import (
	"bytes"
	"sync"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-wallet/key"
)

const round = uint64(1)

type mockSigner struct {
	bus *eventbus.EventBus
}

func (m *mockSigner) Sign(header.Header) ([]byte, error) {
	return make([]byte, 33), nil
}

func (m *mockSigner) SendAuthenticated(topic topics.Topic, hdr header.Header, b *bytes.Buffer, id uint32) error {
	m.bus.Publish(topic, b)
	return nil
}

func (m *mockSigner) SendWithHeader(topic topics.Topic, hash []byte, b *bytes.Buffer, id uint32) error {
	m.bus.Publish(topic, b)
	return nil
}

// FactoryFunc is a shorthand for the reduction factories to create a Reducer
type FactoryFunc func(*eventbus.EventBus, *rpcbus.RPCBus, key.ConsensusKeys, time.Duration) Reducer

// Helper for reducing test boilerplate
type Helper struct {
	Reducer   Reducer
	Bus       *eventbus.EventBus
	RBus      *rpcbus.RPCBus
	Keys      []key.ConsensusKeys
	P         *user.Provisioners
	signer    consensus.Signer
	PubKeyBLS []byte
	nr        int
	Handler   *Handler
	*consensus.SimplePlayer
	CollectionWaitGroup sync.WaitGroup
}

// NewHelper creates a Helper
func NewHelper(eb *eventbus.EventBus, rpcbus *rpcbus.RPCBus, provisioners int, factory FactoryFunc, timeOut time.Duration) *Helper {
	p, keys := consensus.MockProvisioners(provisioners)
	red := factory(eb, rpcbus, keys[0], timeOut)
	hlp := &Helper{
		Bus:          eb,
		RBus:         rpcbus,
		Keys:         keys,
		P:            p,
		Reducer:      red,
		signer:       &mockSigner{eb},
		nr:           provisioners,
		Handler:      NewHandler(keys[0], *p),
		SimplePlayer: consensus.NewSimplePlayer(),
	}

	return hlp
}

// Verify StepVotes. The step must be specified otherwise verification would be dependent on the state of the Helper
func (hlp *Helper) Verify(hash []byte, sv *agreement.StepVotes, step uint8) error {
	vc := hlp.P.CreateVotingCommittee(round, step, hlp.nr)
	sub := vc.IntersectCluster(sv.BitSet)
	apk, err := agreement.ReconstructApk(sub.Set)
	if err != nil {
		return err
	}

	return header.VerifySignatures(round, step, hash, apk, sv.Signature)
}

// SendBatch of consensus events to the reducer callback CollectReductionEvent
func (hlp *Helper) SendBatch(hash []byte) {
	hlp.CollectionWaitGroup.Wait()
	batch := hlp.Spawn(hash)
	hlp.CollectionWaitGroup.Add(len(batch))
	for _, ev := range batch {
		go func(ev consensus.Event) {
			if err := hlp.Reducer.Collect(ev); err != nil {
				panic(err)
			}

			hlp.CollectionWaitGroup.Done()
		}(ev)
	}
}

// Spawn a number of different valid events to the Agreement component bypassing the EventBus
func (hlp *Helper) Spawn(hash []byte) []consensus.Event {
	evs := make([]consensus.Event, 0, hlp.nr)
	step := hlp.Step()
	i := 0
	for count := 0; count < hlp.Handler.Quorum(); {
		ev := MockConsensusEvent(hash, round, step, hlp.Keys, i)
		i++
		evs = append(evs, ev)
		count += hlp.Handler.VotesFor(hlp.Keys[i].BLSPubKeyBytes, round, step)
	}
	return evs
}

// Initialize the reducer with a Round update
func (hlp *Helper) Initialize(ru consensus.RoundUpdate) {
	hlp.Reducer.Initialize(hlp, hlp.signer, ru)
}
