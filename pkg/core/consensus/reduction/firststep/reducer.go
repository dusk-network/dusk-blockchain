package firststep

import (
	"bytes"
	"encoding/hex"
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

var lg = log.WithField("process", "first-step reduction")
var emptyHash = [32]byte{}
var regenerationPackage = new(bytes.Buffer)

// Reducer for the firststep. Although its logic is fairly close to the second step reducer, there are nuances that prevent the use of a generalized Reducer for both steps.
// For instance, the first step reducer produces a StepVote as a result (as opposed to an Agreement), while the start of Reduction event collection should happen after a BestScore event is received (as opposed to a first StepVote)
type Reducer struct {
	broker      eventbus.Broker
	rpcBus      *rpcbus.RPCBus
	keys        key.ConsensusKeys
	eventPlayer consensus.EventPlayer
	signer      consensus.Signer

	reductionID uint32

	handler    *reduction.Handler
	aggregator *aggregator
	timeOut    time.Duration
	Timer      *reduction.Timer
	round      uint64
}

// NewComponent returns an uninitialized reduction component.
func NewComponent(broker eventbus.Broker, rpcBus *rpcbus.RPCBus, keys key.ConsensusKeys, timeOut time.Duration) reduction.Reducer {
	return &Reducer{
		broker:  broker,
		rpcBus:  rpcBus,
		keys:    keys,
		timeOut: timeOut,
	}
}

// Initialize the reduction component, by instantiating the handler and creating
// the topic subscribers.
// Implements consensus.Component
func (r *Reducer) Initialize(eventPlayer consensus.EventPlayer, signer consensus.Signer, ru consensus.RoundUpdate) []consensus.TopicListener {
	r.eventPlayer = eventPlayer
	r.signer = signer
	r.handler = reduction.NewHandler(r.keys, ru.P)
	r.Timer = reduction.NewTimer(r.Halt)
	r.round = ru.Round

	bestScoreSubscriber := consensus.TopicListener{
		Topic:    topics.BestScore,
		Listener: consensus.NewSimpleListener(r.CollectBestScore, consensus.LowPriority, false),
	}

	reductionSubscriber := consensus.TopicListener{
		Topic:         topics.Reduction,
		Preprocessors: []eventbus.Preprocessor{consensus.NewRepublisher(r.broker, topics.Reduction), &consensus.Validator{}},
		Listener:      consensus.NewFilteringListener(r.Collect, r.Filter, consensus.LowPriority, true),
	}
	r.reductionID = reductionSubscriber.Listener.ID()

	return []consensus.TopicListener{bestScoreSubscriber, reductionSubscriber}
}

// ID returns the listener ID of the reducer.
// Implements consensus.Component.
func (r *Reducer) ID() uint32 {
	return r.reductionID
}

// Finalize the Reducer component by killing the timer, and pausing event streaming.
// This will stop a reduction cycle short, and renders this Reducer useless
// after calling.
// Implements consensus.Component.
func (r *Reducer) Finalize() {
	r.eventPlayer.Pause(r.reductionID)
	r.Timer.Stop()
}

// Collect forwards Reduction to the aggregator
func (r *Reducer) Collect(e consensus.Event) error {
	ev := reduction.New()
	if err := reduction.Unmarshal(&e.Payload, ev); err != nil {
		return err
	}

	if err := r.handler.VerifySignature(e.Header, ev.SignedHash); err != nil {
		return err
	}

	lg.WithFields(log.Fields{
		"round":  e.Header.Round,
		"step":   e.Header.Step,
		"sender": hex.EncodeToString(e.Header.Sender()),
		"id":     r.reductionID,
	}).Debugln("received event")
	return r.aggregator.collectVote(*ev, e.Header)
}

// Filter an incoming Reduction message, by checking whether or not it was sent
// by a member of the voting committee for the given round and step.
func (r *Reducer) Filter(hdr header.Header) bool {
	return !r.handler.IsMember(hdr.PubKeyBLS, hdr.Round, hdr.Step)
}

func (r *Reducer) startReduction() {
	r.Timer.Start(r.timeOut)
	r.aggregator = newAggregator(r.Halt, r.handler, r.rpcBus)
}

func (r *Reducer) sendReduction(hash []byte) {
	sig, err := r.signer.Sign(hash, nil)
	if err != nil {
		lg.WithField("category", "BUG").WithError(err).Errorln("error in signing reduction")
		return
	}

	payload := new(bytes.Buffer)
	if err := encoding.WriteBLS(payload, sig); err != nil {
		lg.WithField("category", "BUG").WithError(err).Errorln("error in encoding BLS signature")
		return
	}

	if err := r.signer.SendAuthenticated(topics.Reduction, hash, payload, r.ID()); err != nil {
		lg.WithField("category", "BUG").WithError(err).Errorln("error in sending authenticated Reduction")
	}
}

// Halt will end the first step of reduction, and forwards whatever result it received
// on the StepVotes topic.
func (r *Reducer) Halt(hash []byte, svs ...*agreement.StepVotes) {
	lg.WithField("id", r.reductionID).Traceln("halted")
	r.Timer.Stop()
	r.eventPlayer.Pause(r.reductionID)
	r.timeOut = r.timeOut * 2
	buf := new(bytes.Buffer)
	if len(svs) > 0 {
		if err := agreement.MarshalStepVotes(buf, svs[0]); err != nil {
			lg.WithField("category", "BUG").WithError(err).Errorln("error in marshalling StepVotes")
			return
		}
	}

	r.signer.SendWithHeader(topics.StepVotes, hash, buf, r.ID())
}

// CollectBestScore activates the 2-step reduction cycle.
func (r *Reducer) CollectBestScore(e consensus.Event) error {
	lg.WithField("id", r.reductionID).Traceln("starting reduction")
	r.startReduction()
	step := r.eventPlayer.Forward(r.ID())
	r.eventPlayer.Play(r.reductionID)

	// sending reduction can very well be done concurrently
	if r.handler.AmMember(r.round, step) {
		go r.sendReduction(e.Header.BlockHash)
	}

	return nil
}
