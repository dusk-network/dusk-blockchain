package secondstep

import (
	"bytes"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-wallet/key"
	log "github.com/sirupsen/logrus"
)

var _ consensus.Component = (*reducer)(nil)

var emptyHash = [32]byte{}
var regenerationPackage = new(bytes.Buffer)
var lg = log.WithField("process", "second-step reduction")

type reducer struct {
	broker     eventbus.Broker
	rpcBus     *rpcbus.RPCBus
	keys       key.ConsensusKeys
	stepper    consensus.Stepper
	signer     consensus.Signer
	subscriber consensus.Subscriber

	reductionID uint32

	handler    *reduction.Handler
	aggregator *aggregator
	timeOut    time.Duration
	timer      *reduction.Timer
}

// NewComponent returns an uninitialized reduction component.
func NewComponent(broker eventbus.Broker, rpcBus *rpcbus.RPCBus, keys key.ConsensusKeys, timeOut time.Duration) consensus.Component {
	return &reducer{
		broker:  broker,
		rpcBus:  rpcBus,
		keys:    keys,
		timeOut: timeOut,
	}
}

// Initialize the reduction component, by instantiating the handler and creating
// the topic subscribers.
// Implements consensus.Component
func (r *reducer) Initialize(stepper consensus.Stepper, signer consensus.Signer, ru consensus.RoundUpdate) []consensus.TopicListener {
	r.stepper = stepper
	r.signer = signer
	r.handler = reduction.NewHandler(r.keys, ru.P)
	r.timer = reduction.NewTimer(r.Halt)

	stepVotesSubscriber := consensus.TopicListener{
		Topic:    topics.StepVotes,
		Listener: consensus.NewSimpleListener(r.CollectStepVotes),
	}

	return []consensus.TopicListener{stepVotesSubscriber}
}

// Finalize the reducer component by killing the timer, if it is still running.
// This will stop a reduction cycle short, and renders this reducer useless
// after calling.
func (r *reducer) Finalize() {
	r.timer.Stop()
}

func (r *reducer) CollectReductionEvent(e consensus.Event) error {
	ev := reduction.New()
	if err := reduction.Unmarshal(&e.Payload, ev); err != nil {
		return err
	}

	if err := r.handler.VerifySignature(e.Header, ev.SignedHash); err != nil {
		return err
	}

	r.aggregator.collectVote(*ev, e.Header)
	return nil
}

func (r *reducer) Filter(hdr header.Header) bool {
	return !r.handler.IsMember(hdr.PubKeyBLS, hdr.Round, hdr.Step)
}

func (r *reducer) startReduction(sv *agreement.StepVotes) {
	r.timer.Start(r.timeOut)
	r.aggregator = newAggregator(r.Halt, r.handler, sv)
}

func (r *reducer) sendReduction(hash []byte) error {
	sig, err := r.signer.Sign(hash, nil)
	if err != nil {
		return err
	}

	payload := new(bytes.Buffer)
	if err := encoding.WriteBLS(payload, sig); err != nil {
		return err
	}

	return r.signer.SendAuthenticated(topics.Reduction, hash, payload)
}

// Halt is used by either the Aggregator in case of succesful reduction or the timer in case of a timeout.
// In the latter case no agreement message is pushed forward
func (r *reducer) Halt(hash []byte, b ...*agreement.StepVotes) {
	r.subscriber.Unsubscribe(r.reductionID)
	r.broker.Publish(topics.Regeneration, nil)

	// TODO: check if an agreement on an empty block should be propagated
	if hash != nil && len(b) > 0 {
		r.sendAgreement(hash, b)
	}
	r.stepper.RequestStepUpdate()
}

// CollectStepVotes is triggered when the first StepVotes get published by the first step reducer
func (r *reducer) CollectStepVotes(e consensus.Event) error {
	listener := consensus.NewFilteringListener(r.CollectReductionEvent, r.Filter)

	r.subscriber.Subscribe(topics.Reduction, listener)
	r.reductionID = listener.ID()

	sv, err := agreement.UnmarshalStepVotes(&e.Payload)
	if err != nil {
		return err
	}
	r.sendReduction(e.Header.BlockHash)
	r.startReduction(sv)
	return nil
}

func (r *reducer) sendAgreement(hash []byte, svs []*agreement.StepVotes) {
	payloadBuf := new(bytes.Buffer)
	if err := agreement.MarshalVotes(payloadBuf, svs); err != nil {
		lg.WithField("category", "BUG").WithError(err).Errorln("cannot marshal the StepVotes")
		return
	}

	sig, err := r.signer.Sign(hash, payloadBuf.Bytes())
	if err != nil {
		lg.WithField("category", "BUG").WithError(err).Errorln("cannot sign the agreement")
		return
	}

	// then we create the full BLS signed Agreement
	ev := agreement.Agreement{}
	ev.SetSignature(sig)
	ev.VotesPerStep = svs

	eventBuf := new(bytes.Buffer)
	if err := agreement.Marshal(eventBuf, ev); err != nil {
		lg.WithField("category", "BUG").WithError(err).Errorln("cannot marshal the agreement")
		return
	}

	// then we forward the marshalled Agreement to the store to be sent
	if err := r.signer.SendAuthenticated(topics.Agreement, hash, eventBuf); err != nil {
		lg.WithField("category", "BUG").WithError(err).Errorln("error in Ed25519 signing and gossip the agreement")
	}
}
