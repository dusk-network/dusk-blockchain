package firststep

import (
	"encoding/hex"
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
	Timer      *reduction.Timer
	round      uint64
}

// NewComponent returns an uninitialized reduction component.
func NewComponent(broker eventbus.Broker, rpcBus *rpcbus.RPCBus, keys key.Keys, timeOut time.Duration) reduction.Reducer {
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
		Topic:    topics.Reduction,
		Listener: consensus.NewFilteringListener(r.Collect, r.Filter, consensus.LowPriority, true),
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
func (r *Reducer) Collect(e consensus.InternalPacket) error {
	red := e.(message.Reduction)

	if err := r.handler.VerifySignature(red); err != nil {
		return err
	}

	hdr := red.State()
	lg.WithFields(log.Fields{
		"round":  hdr.Round,
		"step":   hdr.Step,
		"sender": hex.EncodeToString(hdr.Sender()),
		"id":     r.reductionID,
		"hash":   hex.EncodeToString(hdr.BlockHash),
	}).Debugln("received event")
	return r.aggregator.collectVote(red)
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

func (r *Reducer) sendReduction(step uint8, hash []byte) {
	hdr := r.constructHeader(step, hash)
	sig, err := r.signer.Sign(hdr)
	if err != nil {
		lg.WithField("category", "BUG").WithError(err).Errorln("error in signing reduction")
		return
	}
	red := message.NewReduction(hdr)
	red.SignedHash = sig
	msg := message.New(topics.Reduction, *red)

	if err := r.signer.Gossip(msg, r.ID()); err != nil {
		lg.WithField("category", "BUG").WithError(err).Errorln("error in sending authenticated Reduction")
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
func (r *Reducer) Halt(hash []byte, svs ...*message.StepVotes) {
	var svm message.StepVotesMsg
	lg.WithField("id", r.reductionID).Traceln("halted")
	r.Timer.Stop()
	r.eventPlayer.Pause(r.reductionID)

	if len(svs) > 0 {
		sv := *svs[0]
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
			lg.WithField("timeout", r.timeOut).Trace("max_timeout_reached")
			r.timeOut = 60 * time.Second
		}
		lg.WithField("timeout", r.timeOut).Trace("increase_timeout")
		svm = r.signer.Compose(factory).(message.StepVotesMsg)
	}

	msg := message.New(topics.StepVotes, svm)
	_ = r.signer.SendInternally(topics.StepVotes, msg, r.ID())
}

// CollectBestScore activates the 2-step reduction cycle.
// TODO: interface - rename into CollectStartReductionSignal
func (r *Reducer) CollectBestScore(e consensus.InternalPacket) error {
	lg.WithField("id", r.reductionID).Traceln("starting reduction")
	r.startReduction()
	step := r.eventPlayer.Forward(r.ID())
	r.eventPlayer.Play(r.reductionID)

	if r.handler.AmMember(r.round, step) {
		hash := e.State().BlockHash
		r.sendReduction(step, hash)
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
