package reduction

import (
	"sync"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/key"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
)

const round = uint64(1)

type mockSigner struct {
	bus    *eventbus.EventBus
	pubkey []byte
}

func (m *mockSigner) Sign(header.Header) ([]byte, error) {
	return make([]byte, 33), nil
}

func (m *mockSigner) Compose(pf consensus.PacketFactory) consensus.InternalPacket {
	return pf.Create(m.pubkey, 1, 1)
}

func (m *mockSigner) Gossip(msg message.Message, id uint32) error {
	m.bus.Publish(msg.Category(), msg)
	return nil
}

func (m *mockSigner) SendInternally(topic topics.Topic, msg message.Message, id uint32) error {
	m.bus.Publish(topic, msg)
	return nil
}

// FactoryFunc is a shorthand for the reduction factories to create a Reducer
type FactoryFunc func(*eventbus.EventBus, *rpcbus.RPCBus, key.ConsensusKeys, time.Duration) Reducer

// Helper for reducing test boilerplate
type Helper struct {
	PubKeyBLS []byte
	Reducer   Reducer
	Bus       *eventbus.EventBus
	RBus      *rpcbus.RPCBus
	Keys      []key.ConsensusKeys
	P         *user.Provisioners
	signer    consensus.Signer
	nr        int
	Handler   *Handler
	*consensus.SimplePlayer
	CollectionWaitGroup sync.WaitGroup
}

// NewHelper creates a Helper
func NewHelper(eb *eventbus.EventBus, rpcbus *rpcbus.RPCBus, provisioners int, factory FactoryFunc, timeOut time.Duration) *Helper {
	p, keys := consensus.MockProvisioners(provisioners)
	helperKeys := keys[0]
	red := factory(eb, rpcbus, helperKeys, timeOut)
	hlp := &Helper{
		PubKeyBLS: helperKeys.BLSPubKeyBytes,
		Bus:       eb,
		RBus:      rpcbus,
		Keys:      keys,
		P:         p,
		Reducer:   red,

		signer:       &mockSigner{eb, helperKeys.BLSPubKeyBytes},
		nr:           provisioners,
		Handler:      NewHandler(helperKeys, *p),
		SimplePlayer: consensus.NewSimplePlayer(),
	}

	return hlp
}

// Verify StepVotes. The step must be specified otherwise verification would be dependent on the state of the Helper
func (hlp *Helper) Verify(hash []byte, sv message.StepVotes, step uint8) error {
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
	// creating a batch of Reduction events
	batch := hlp.Spawn(hash)
	hlp.CollectionWaitGroup.Add(len(batch))
	for _, ev := range batch {
		go func(ev consensus.Packet) {
			if err := hlp.Reducer.Collect(ev); err != nil {
				panic(err)
			}

			hlp.CollectionWaitGroup.Done()
		}(ev)
	}
}

// Spawn a number of different valid events to the Agreement component bypassing the EventBus
func (hlp *Helper) Spawn(hash []byte) []message.Reduction {
	evs := make([]message.Reduction, 0, hlp.nr)
	step := hlp.Step()
	i := 0
	for count := 0; count < hlp.Handler.Quorum(hlp.Round); {
		ev := message.MockReduction(hash, round, step, hlp.Keys, i)
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
