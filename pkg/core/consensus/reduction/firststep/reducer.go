package firststep

import (
	"sync/atomic"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	log "github.com/sirupsen/logrus"
)

var _ consensus.Component = (*Reducer)(nil)

var lg = log.WithField("process", "first-step reduction")
var emptyHash = [32]byte{}

// Reducer for the firststep. Although its logic is fairly close to the second step reducer, there are nuances that prevent the use of a generalized Reducer for both steps.
// For instance, the first step reducer produces a StepVote as a result (as opposed to an Agreement), while the start of Reduction event collection should happen after a BestScore event is received (as opposed to a first StepVote)
type Reducer struct {
	broker      eventbus.Broker
	rpcBus      *rpcbus.RPCBus
	keys        key.Keys
	eventPlayer consensus.EventPlayer
	signer      consensus.Signer

	reductionID uint32

	handler    *reduction.Handler
	aggregator *aggregator
	timeOut    time.Duration
	timer      *reduction.Timer
	round      uint64

	haltChan chan reduction.HaltMsg
	quitChan chan struct{}
}

// NewComponent returns an uninitialized reduction component.
func NewComponent(broker eventbus.Broker, rpcBus *rpcbus.RPCBus, keys key.Keys, timeOut time.Duration) reduction.Reducer {
	return &Reducer{
		broker:   broker,
		rpcBus:   rpcBus,
		keys:     keys,
		timeOut:  timeOut,
		haltChan: make(chan reduction.HaltMsg, 2),
		quitChan: make(chan struct{}, 1),
	}
}

// Initialize the reduction component, by instantiating the handler and creating
// the topic subscribers.
// Implements consensus.Component
func (r *Reducer) Initialize(eventPlayer consensus.EventPlayer, signer consensus.Signer, ru consensus.RoundUpdate) []consensus.TopicListener {
	r.eventPlayer = eventPlayer
	r.signer = signer
	r.handler = reduction.NewHandler(r.keys, ru.P)
	r.timer = reduction.NewTimer(r.haltChan)
	r.round = ru.Round

	bestScoreSubscriber := consensus.TopicListener{
		Topic:    topics.BestScore,
		Listener: consensus.NewSimpleListener(r.CollectBestScore, consensus.LowPriority, false),
	}

	reductionSubscriber := consensus.TopicListener{
		Topic:    topics.Reduction,
		Listener: consensus.NewFilteringListener(r.Collect, r.Filter, consensus.LowPriority, true),
	}

	atomic.StoreUint32(&r.reductionID, reductionSubscriber.Listener.ID())

	return []consensus.TopicListener{bestScoreSubscriber, reductionSubscriber}
}

// ID returns the listener ID of the reducer.
// Implements consensus.Component.
func (r *Reducer) ID() uint32 {
	return atomic.LoadUint32(&r.reductionID)
}

// Name returns the listener Name of the reducer.
// Implements consensus.Component.
func (r *Reducer) Name() string {
	return "firststep/Reducer"
}

// Finalize the Reducer component by killing the timer, and pausing event streaming.
// This will stop a reduction cycle short, and renders this Reducer useless
// after calling.
// Implements consensus.Component.
func (r *Reducer) Finalize() {
	r.eventPlayer.Pause(r.reductionID)
	r.timer.Stop()
	r.quitChan <- struct{}{}
}

// Collect forwards Reduction to the aggregator
func (r *Reducer) Collect(e consensus.InternalPacket) error {
	reductionMessage := e.(message.Reduction)

	if err := r.handler.VerifySignature(reductionMessage); err != nil {
		return err
	}

	hdr := reductionMessage.State()
	lg.WithFields(log.Fields{
		"round": hdr.Round,
		"step":  hdr.Step,
		//"sender": hex.EncodeToString(hdr.Sender()),
		"id": r.reductionID,
		//"hash":   hex.EncodeToString(hdr.BlockHash),
	}).Debugln("received_event")
	return r.aggregator.collectVote(reductionMessage)
}

// Filter an incoming Reduction message, by checking whether or not it was sent
// by a member of the voting committee for the given round and step.
func (r *Reducer) Filter(hdr header.Header) bool {
	return !r.handler.IsMember(hdr.PubKeyBLS, hdr.Round, hdr.Step)
}

func (r *Reducer) startReduction() {
	lg.WithFields(log.Fields{
		"round":   r.round,
		"id":      r.reductionID,
		"timeout": r.timeOut / time.Second,
	}).Debugln("startReduction")

	// Clear out haltChan
	for len(r.haltChan) > 0 {
		<-r.haltChan
	}

	go r.collectHalt()
	r.timer.Start(r.timeOut)
	r.aggregator = newAggregator(r.haltChan, r.handler, r.rpcBus)
}

func (r *Reducer) sendReduction(step uint8, hash []byte) error {
	hdr := r.constructHeader(step, hash)
	sig, err := r.signer.Sign(hdr)
	if err != nil {
		lg.WithField("category", "BUG").WithError(err).Errorln("error in signing reduction")
		return err
	}
	red := message.NewReduction(hdr)
	red.SignedHash = sig
	msg := message.New(topics.Reduction, *red)

	if err := r.signer.Gossip(msg, r.ID()); err != nil {
		lg.WithField("category", "BUG").WithError(err).Errorln("error in sending authenticated Reduction")
		return err
	}
	return nil
}

// collectHalt is called by the Aggregator and the Timer through a channel, and ensures that only
// one `Halt` call is made per step.
func (r *Reducer) collectHalt() {
	select {
	case m := <-r.haltChan:
		r.Halt(m.Hash, m.Sv)
	case <-r.quitChan:
	}
}

// StepVotesMsgFactory creates a StepVotesMsg to be passed internally within
// the Consensus components
type StepVotesMsgFactory struct {
	sv   message.StepVotes
	hash []byte
}

// Create a new StepVotesMsgFactory
func (s StepVotesMsgFactory) Create(sender []byte, round uint64, step uint8) consensus.InternalPacket {
	// handling the case if the StepVotes is empty
	// TODO: interface - check this. The step sounds like it is intrinsic
	// within the StepVotes, but also (before the refactoring) the Header was
	// constructed by the Coordinator
	if s.sv.Step == 0 {
		s.sv.Step = step
	}
	return message.NewStepVotesMsg(round, s.hash, sender, s.sv)
}

// Halt will end the first step of reduction, and forwards whatever result it received
// on the StepVotes topic.
func (r *Reducer) Halt(hash []byte, stepVotes []*message.StepVotes) {
	var svm message.StepVotesMsg
	lg.
		WithField("id", r.reductionID).
		WithField("round", r.round).
		Traceln("firststep_halted")
	r.timer.Stop()
	r.eventPlayer.Pause(r.reductionID)

	if len(stepVotes) > 0 {
		sv := *stepVotes[0]
		svm = message.NewStepVotesMsg(r.round, hash, r.keys.BLSPubKeyBytes, sv)
	} else {
		//create a factory for empty StepVotes (necessary since we cannot get
		//the Step from the StepVotes and we likely need a valid Header)
		factory := StepVotesMsgFactory{
			sv:   message.StepVotes{},
			hash: hash,
		}
		// Increase timeout if we did not have a good result
		r.timeOut = r.timeOut * 2
		if r.timeOut > 60*time.Second {
			lg.
				WithField("timeout", r.timeOut).
				WithField("id", r.reductionID).
				WithField("round", r.round).
				Error("max_timeout_reached")
			r.timeOut = 60 * time.Second
		}
		lg.WithField("timeout", r.timeOut).
			WithField("id", r.reductionID).
			WithField("round", r.round).
			Trace("increase_timeout")
		svm = r.signer.Compose(factory).(message.StepVotesMsg)
	}

	msg := message.New(topics.StepVotes, svm)
	err := r.signer.SendInternally(topics.StepVotes, msg, r.ID())
	if err != nil {
		lg.WithField("timeout", r.timeOut).
			WithField("id", r.reductionID).
			WithField("round", r.round).
			Error("firststep_halted, failed to SendInternally, topics.StepVotes")
		// FIXME: shall this panic ? is this a extreme violation ?
		//panic("could not SendInternally, topics.StepVotes")
	}
}

// CollectBestScore activates the 2-step reduction cycle.
// TODO: interface - rename into CollectStartReductionSignal
func (r *Reducer) CollectBestScore(e consensus.InternalPacket) error {
	hdr := e.State()
	r.startReduction()
	// fetch the right step
	step := r.eventPlayer.Forward(r.ID())
	r.eventPlayer.Play(r.reductionID)

	lg.
		WithField("id", r.reductionID).
		WithField("round", r.round).
		WithField("step", step).
		Trace("starting firststep reduction")

	if r.handler.AmMember(r.round, step) {
		// propagating a new StepVoteMsg with the right step
		hash := hdr.BlockHash
		return r.sendReduction(step, hash)
	}
	return nil
}

func (r *Reducer) constructHeader(step uint8, hash []byte) header.Header {
	return header.Header{
		Round:     r.round,
		Step:      step,
		PubKeyBLS: r.keys.BLSPubKeyBytes,
		BlockHash: hash,
	}
}
