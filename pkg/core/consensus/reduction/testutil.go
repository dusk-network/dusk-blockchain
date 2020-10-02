package reduction

import (
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
)

// Helper for reducing test boilerplate
type Helper struct {
	PubKeyBLS []byte
	Bus       *eventbus.EventBus
	RBus      *rpcbus.RPCBus
	Keys      []key.Keys
	P         *user.Provisioners
	nr        int
	Handler   *Handler
	Emitter   *consensus.Emitter
}

// NewHelper creates a Helper
func NewHelper(provisioners int, timeOut time.Duration) *Helper {
	p, keys := consensus.MockProvisioners(provisioners)

	mockProxy := transactions.MockProxy{
		P: transactions.PermissiveProvisioner{},
	}
	emitter := consensus.MockEmitter(timeOut, mockProxy)
	emitter.Keys = keys[0]

	hlp := &Helper{
		PubKeyBLS: emitter.Keys.BLSPubKeyBytes,
		Bus:       emitter.EventBus,
		RBus:      emitter.RPCBus,
		Keys:      keys,
		P:         p,
		nr:        provisioners,
		Handler:   NewHandler(emitter.Keys, *p),
		Emitter:   emitter,
	}

	return hlp
}

// Verify StepVotes. The step must be specified otherwise verification would be dependent on the state of the Helper
func (hlp *Helper) Verify(hash []byte, sv message.StepVotes, round uint64, step uint8) error {
	vc := hlp.P.CreateVotingCommittee(round, step, hlp.nr)
	sub := vc.IntersectCluster(sv.BitSet)
	apk, err := agreement.ReconstructApk(sub.Set)
	if err != nil {
		return err
	}

	return header.VerifySignatures(round, step, hash, apk, sv.Signature)
}

// Spawn a number of different valid events to the Agreement component bypassing the EventBus
func (hlp *Helper) Spawn(hash []byte, round uint64, step uint8) []message.Reduction {
	evs := make([]message.Reduction, 0, hlp.nr)
	i := 0
	for count := 0; count < hlp.Handler.Quorum(round); {
		ev := message.MockReduction(hash, round, step, hlp.Keys, i)
		i++
		evs = append(evs, ev)
		count += hlp.Handler.VotesFor(hlp.Keys[i].BLSPubKeyBytes, round, step)
	}
	return evs
}
