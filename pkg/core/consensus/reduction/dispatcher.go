package reduction

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

type voteDispatcher struct {
	firstStepVoteSet *agreement.StepVotes
	publisher        eventbus.Publisher
	s                *consensus.Signer
}

func (v *voteDispatcher) collectStepVotes(sv agreement.StepVotes) error {
	if v.firstStepVoteSet == nil {
		v.firstStepVoteSet = &sv
		return nil
	}

	sig, err := v.signVotes(sv)
	if err != nil {
		return err
	}

	vote, err := v.generateAgreement(sv, sig)
	if err != nil {
		return err
	}

	v.publisher.Publish(topics.Gossip, vote)
	v.clear()
	return nil
}

func (v *voteDispatcher) clear() {
	v.firstStepVoteSet = nil
}

func (v *voteDispatcher) signVotes(sv agreement.StepVotes) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := agreement.MarshalVotes(buf, []agreement.StepVotes{*v.firstStepVoteSet, sv}); err != nil {
		return nil, err
	}

	return v.s.BLSSign(buf.Bytes())
}

func (v *voteDispatcher) generateAgreement(sv agreement.StepVotes, sig []byte) (*bytes.Buffer, error) {
	ev := agreement.New()
	ev.SignedVotes = sig
	ev.VotesPerStep = []agreement.StepVotes{*v.firstStepVoteSet, sv}

	buf := new(bytes.Buffer)
	if err := agreement.Marshal(buf, ev); err != nil {
		return nil, err
	}

	return buf, nil
}
