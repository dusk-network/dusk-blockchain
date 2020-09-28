package secondstep

import (
	"bytes"
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

var lg = log.WithField("process", "second-step reduction")
var emptyHash = [32]byte{}
var restartFactory = consensus.Restarter{}

// Reducer for the second step. This reducer starts whenever it receives an internal
// StepVotes message. It combines the contents of this message (if any) with the
// result of it's own reduction step, and on success, creates and sends an Agreement
// message.
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

	stepVotesSubscriber := consensus.TopicListener{
		Topic:    topics.StepVotes,
		Listener: consensus.NewSimpleListener(r.CollectStepVotes, consensus.LowPriority, false),
	}

	reductionSubscriber := consensus.TopicListener{
		Topic:    topics.Reduction,
		Listener: consensus.NewFilteringListener(r.Collect, r.Filter, consensus.LowPriority, true),
	}
	atomic.StoreUint32(&r.reductionID, reductionSubscriber.Listener.ID())

	return []consensus.TopicListener{stepVotesSubscriber, reductionSubscriber}
}

// ID returns the listener ID of the reducer.
// Implements consensus.Component.
func (r *Reducer) ID() uint32 {
	return atomic.LoadUint32(&r.reductionID)
}

// Name returns the listener Name of the reducer.
// Implements consensus.Component.
func (r *Reducer) Name() string {
	return "secondstep/Reducer"
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

// Collect complies with the consensus.Component interface. It gathers
// a message.Reduction, verifies its signature and forwards it to the
// aggregator
func (r *Reducer) Collect(e consensus.InternalPacket) error {
	reductionMessage := e.(message.Reduction)
	hdr := reductionMessage.State()

	if err := r.handler.VerifySignature(reductionMessage); err != nil {
		return err
	}

	lg.WithFields(log.Fields{
		"round": hdr.Round,
		"step":  hdr.Step,
		//"sender": hex.EncodeToString(ev.Sender()),
		"id": r.reductionID,
		//"hash": hex.EncodeToString(hdr.BlockHash),
	}).Debugln("received_event")
	return r.aggregator.collectVote(reductionMessage)
}

// Filter an incoming Reduction message, by checking whether or not it was sent
// by a member of the voting committee for the given round and step.
func (r *Reducer) Filter(hdr header.Header) bool {
	return !r.handler.IsMember(hdr.PubKeyBLS, hdr.Round, hdr.Step)
}

func (r *Reducer) startReduction(firstStepVotes message.StepVotesMsg) {
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
	r.aggregator = newAggregator(r.haltChan, r.handler, &firstStepVotes.StepVotes)
}

// sendReduction creates a new Reduction message and Gossip it externally
// TODO: interface - consider moving this function into the reduction/reducer
// package
func (r *Reducer) sendReduction(step uint8, hash []byte) error {
	hdr := r.constructHeader(step, hash)
	sig, err := r.signer.Sign(hdr)
	if err != nil {
		return err
	}
	red := message.NewReduction(hdr)
	red.SignedHash = sig
	msg := message.New(topics.Reduction, *red)

	if err := r.signer.Gossip(msg, r.ID()); err != nil {
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

// Halt is used by either the Aggregator in case of successful reduction or the timer in case of a timeout.
// In the latter case no agreement message is pushed forward
func (r *Reducer) Halt(hash []byte, stepVotes []*message.StepVotes) {
	lg.
		WithField("len_step", len(stepVotes)).
		WithField("id", r.reductionID).
		WithField("round", r.round).
		Trace("secondstep_halted")
	r.timer.Stop()
	r.eventPlayer.Pause(r.reductionID)

	// Sending of agreement happens on it's own step
	step := r.eventPlayer.Forward(r.ID())
	if hash != nil && !bytes.Equal(hash, emptyHash[:]) && stepVotesAreValid(stepVotes) && r.handler.AmMember(r.round, step) {
		lg.
			WithField("step", step).
			WithField("id", r.reductionID).
			WithField("round", r.round).
			Debug("sending agreement")
		r.sendAgreement(step, hash, stepVotes)
	} else {
		// Increase timeout if we had no agreement
		r.timeOut = r.timeOut * 2
		if r.timeOut > 60*time.Second {
			lg.
				WithField("timeout", r.timeOut).
				WithField("step", step).
				WithField("id", r.reductionID).
				WithField("round", r.round).
				Error("max_timeout_reached")
			r.timeOut = 60 * time.Second
		}
		lg.WithField("timeout", r.timeOut).
			WithField("step", step).
			WithField("id", r.reductionID).
			WithField("round", r.round).
			Trace("increase_timeout")
	}

	restart := r.signer.Compose(restartFactory)
	msg := message.New(topics.Restart, restart)

	if err := r.signer.SendInternally(topics.Restart, msg, r.ID()); err != nil {
		lg.WithError(err).
			WithField("step", step).
			WithField("id", r.reductionID).
			WithField("round", r.round).
			Error("secondstep_halted, failed to SendInternally, topics.Restart")
		// FIXME: shall this panic ? is this a extreme violation ?
		//panic("could not restart consensus round")
	}
}

// CollectStepVotes is triggered when the first StepVotes get published by the
// first step Reducer, and starts the second step of reduction.
// If the first step did not have a winning block, we should get an empty
// StepVotesMsg and run with it anyway (to keep the security assumptions of the
// protocol right).
func (r *Reducer) CollectStepVotes(e consensus.InternalPacket) error {
	firstStepVotes := e.(message.StepVotesMsg)
	hdr := firstStepVotes.State()
	r.startReduction(firstStepVotes)
	// fetch the right step
	step := r.eventPlayer.Forward(r.ID())
	r.eventPlayer.Play(r.reductionID)

	lg.
		WithField("id", r.reductionID).
		WithField("round", r.round).
		WithField("step", step).
		Trace("starting secondstep reduction")

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

func (r *Reducer) sendAgreement(step uint8, hash []byte, svs []*message.StepVotes) {
	hdr := r.constructHeader(step, hash)
	sig, err := r.signer.Sign(hdr)
	if err != nil {
		lg.WithField("category", "BUG").WithError(err).Error("cannot sign the agreement")
		return
	}

	// then we create the full BLS signed Agreement
	// XXX: the StepVotes are NOT signed (i.e. the message.SignAgreement is not used).
	// This exposes the Agreement to some malleability attack. Double check
	// this!!
	ev := message.NewAgreement(hdr)
	ev.SetSignature(sig)
	ev.VotesPerStep = svs

	msg := message.New(topics.Agreement, *ev)
	// then we forward the marshaled Agreement to the store to be sent
	if err := r.signer.Gossip(msg, r.ID()); err != nil {
		lg.WithField("category", "BUG").WithError(err).Error("error in gossiping the agreement")
	}
}

func stepVotesAreValid(svs []*message.StepVotes) bool {
	return len(svs) == 2 && !svs[0].IsEmpty() && !svs[1].IsEmpty()
}
