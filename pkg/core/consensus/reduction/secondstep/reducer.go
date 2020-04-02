package secondstep

import (
	"bytes"
	"encoding/hex"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-wallet/v2/key"
	log "github.com/sirupsen/logrus"
)

var _ consensus.Component = (*Reducer)(nil)

var emptyHash = [32]byte{}
var lg = log.WithField("process", "second-step reduction")

// Reducer for the second step. This reducer starts whenever it receives an internal
// StepVotes message. It combines the contents of this message (if any) with the
// result of it's own reduction step, and on success, creates and sends an Agreement
// message.
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
	timer      *reduction.Timer
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
	r.timer = reduction.NewTimer(r.Halt)
	r.round = ru.Round

	stepVotesSubscriber := consensus.TopicListener{
		Topic:    topics.StepVotes,
		Listener: consensus.NewSimpleListener(r.CollectStepVotes, consensus.LowPriority, false),
	}

	reductionSubscriber := consensus.TopicListener{
		Topic:    topics.Reduction,
		Listener: consensus.NewFilteringListener(r.Collect, r.Filter, consensus.LowPriority, true),
	}
	r.reductionID = reductionSubscriber.Listener.ID()

	return []consensus.TopicListener{stepVotesSubscriber, reductionSubscriber}
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
	r.timer.Stop()
}

func (r *Reducer) Collect(e consensus.InternalPacket) error {
	ev := e.(message.Reduction)
	hdr := ev.State()

	if err := r.handler.VerifySignature(ev); err != nil {
		return err
	}

	lg.WithFields(log.Fields{
		"round":  hdr.Round,
		"step":   hdr.Step,
		"sender": hex.EncodeToString(ev.Sender()),
		"id":     r.reductionID,
		"hash":   hex.EncodeToString(hdr.BlockHash),
	}).Debugln("received event")
	return r.aggregator.collectVote(ev)
}

// Filter an incoming Reduction message, by checking whether or not it was sent
// by a member of the voting committee for the given round and step.
func (r *Reducer) Filter(hdr header.Header) bool {
	return !r.handler.IsMember(hdr.PubKeyBLS, hdr.Round, hdr.Step)
}

func (r *Reducer) startReduction(sv message.StepVotesMsg) {
	r.timer.Start(r.timeOut)
	r.aggregator = newAggregator(r.Halt, r.handler, &sv.StepVotes)
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

var restartFactory = consensus.Restarter{}

// Halt is used by either the Aggregator in case of successful reduction or the timer in case of a timeout.
// In the latter case no agreement message is pushed forward
func (r *Reducer) Halt(hash []byte, b ...*message.StepVotes) {
	lg.WithField("id", r.reductionID).Traceln("halted")
	r.timer.Stop()
	r.eventPlayer.Pause(r.reductionID)

	// Sending of agreement happens on it's own step
	step := r.eventPlayer.Forward(r.ID())
	if hash != nil && !bytes.Equal(hash, emptyHash[:]) && stepVotesAreValid(b) && r.handler.AmMember(r.round, step) {
		lg.WithField("step", step).Debugln("sending agreement")
		r.sendAgreement(step, hash, b)
	} else {
		// Increase timeout if we had no agreement
		r.timeOut = r.timeOut * 2
	}

	restart := r.signer.Compose(restartFactory)
	msg := message.New(topics.Restart, restart)

	r.signer.SendInternally(topics.Restart, msg, r.ID())
}

// CollectStepVotes is triggered when the first StepVotes get published by the
// first step Reducer, and starts the second step of reduction.
// If the first step did not have a winning block, we should get an empty
// StepVotesMsg and run with it anyway (to keep the security assumptions of the
// protocol right).
func (r *Reducer) CollectStepVotes(e consensus.InternalPacket) error {
	lg.WithField("id", r.reductionID).Traceln("starting reduction")
	sv := e.(message.StepVotesMsg)
	hdr := sv.State()
	r.startReduction(sv)
	// fetch the right step
	step := r.eventPlayer.Forward(r.ID())
	r.eventPlayer.Play(r.reductionID)

	if r.handler.AmMember(r.round, step) {
		// propagating a new StepVoteMsg with the right step
		r.sendReduction(step, hdr.BlockHash)
	}
	return nil
}

func (r *Reducer) sendAgreement(step uint8, hash []byte, svs []*message.StepVotes) {
	hdr := r.constructHeader(step, hash)
	sig, err := r.signer.Sign(hdr)
	if err != nil {
		lg.WithField("category", "BUG").WithError(err).Errorln("cannot sign the agreement")
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
	// then we forward the marshalled Agreement to the store to be sent
	if err := r.signer.Gossip(msg, r.ID()); err != nil {
		lg.WithField("category", "BUG").WithError(err).Errorln("error in gossiping the agreement")
	}
}

func (r *Reducer) constructHeader(step uint8, hash []byte) header.Header {
	return header.Header{
		Round:     r.round,
		Step:      step,
		PubKeyBLS: r.keys.BLSPubKeyBytes,
		BlockHash: hash,
	}
}

func stepVotesAreValid(svs []*message.StepVotes) bool {
	return len(svs) == 2 && !svs[0].IsEmpty() && !svs[1].IsEmpty()
}
