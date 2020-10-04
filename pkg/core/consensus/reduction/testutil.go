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
)

// Helper for reducing test boilerplate
type Helper struct {
	*consensus.Emitter
	ThisSender       []byte
	ProvisionersKeys []key.Keys
	P                *user.Provisioners
	Nr               int
	Handler          *Handler
}

// NewHelper creates a Helper
func NewHelper(provisioners int, timeOut time.Duration) *Helper {
	p, provisionersKeys := consensus.MockProvisioners(provisioners)

	mockProxy := transactions.MockProxy{
		P: transactions.PermissiveProvisioner{},
	}
	emitter := consensus.MockEmitter(timeOut, mockProxy)
	emitter.Keys = provisionersKeys[0]

	hlp := &Helper{
		ThisSender:       emitter.Keys.BLSPubKeyBytes,
		ProvisionersKeys: provisionersKeys,
		P:                p,
		Nr:               provisioners,
		Handler:          NewHandler(emitter.Keys, *p),
		Emitter:          emitter,
	}

	return hlp
}

// Verify StepVotes. The step must be specified otherwise verification would be dependent on the state of the Helper
func (hlp *Helper) Verify(hash []byte, sv message.StepVotes, round uint64, step uint8) error {
	vc := hlp.P.CreateVotingCommittee(round, step, hlp.Nr)
	sub := vc.IntersectCluster(sv.BitSet)
	apk, err := agreement.ReconstructApk(sub.Set)
	if err != nil {
		return err
	}

	return header.VerifySignatures(round, step, hash, apk, sv.Signature)
}

// Spawn a number of different valid events to the Agreement component bypassing the EventBus
func (hlp *Helper) Spawn(hash []byte, round uint64, step uint8) []message.Reduction {
	evs := make([]message.Reduction, 0, hlp.Nr)
	i := 0
	for count := 0; count < hlp.Handler.Quorum(round); {
		ev := message.MockReduction(hash, round, step, hlp.ProvisionersKeys, i)
		i++
		evs = append(evs, ev)
		count += hlp.Handler.VotesFor(hlp.ProvisionersKeys[i].BLSPubKeyBytes, round, step)
	}
	return evs
}
