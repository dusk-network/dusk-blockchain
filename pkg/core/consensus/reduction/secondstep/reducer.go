package secondstep

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

var _ consensus.Component = (*Reducer)(nil)

var emptyHash = [32]byte{}
var regenerationPackage = new(bytes.Buffer)
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

func (r *Reducer) startReduction(sv *agreement.StepVotes) {
	r.timer.Start(r.timeOut)
	r.aggregator = newAggregator(r.Halt, r.handler, sv)
}

func (r *Reducer) sendReduction(hash []byte) error {
	sig, err := r.signer.Sign(hash, nil)
	if err != nil {
		return err
	}

	payload := new(bytes.Buffer)
	if err := encoding.WriteBLS(payload, sig); err != nil {
		return err
	}

	return r.signer.SendAuthenticated(topics.Reduction, hash, payload, r.ID())
}

// Halt is used by either the Aggregator in case of succesful reduction or the timer in case of a timeout.
// In the latter case no agreement message is pushed forward
func (r *Reducer) Halt(hash []byte, b ...*agreement.StepVotes) {
	lg.WithField("id", r.reductionID).Traceln("halted")
	r.timer.Stop()
	r.eventPlayer.Pause(r.reductionID)
	r.timeOut = r.timeOut * 2

	// Sending of agreement happens on it's own step
	step := r.eventPlayer.Forward(r.ID())
	if hash != nil && !bytes.Equal(hash, emptyHash[:]) && stepVotesAreValid(b) && r.handler.AmMember(r.round, step) {
		lg.WithField("step", step).Debugln("sending agreement")
		r.sendAgreement(hash, b)
	}

	r.signer.SendWithHeader(topics.Restart, emptyHash[:], regenerationPackage, r.ID())
}

// CollectStepVotes is triggered when the first StepVotes get published by the
// first step Reducer, and starts the second step of reduction.
func (r *Reducer) CollectStepVotes(e consensus.Event) error {
	lg.WithField("id", r.reductionID).Traceln("starting reduction")
	var sv *agreement.StepVotes

	// If the first step did not have a winning block, we should get an empty buffer
	if e.Payload.Len() > 0 {
		// Otherwise though, we should retrieve the information
		var err error
		sv, err = agreement.UnmarshalStepVotes(&e.Payload)
		if err != nil {
			return err
		}
	}

	r.startReduction(sv)
	step := r.eventPlayer.Forward(r.ID())
	r.eventPlayer.Play(r.reductionID)
	if r.handler.AmMember(r.round, step) {
		go r.sendReduction(e.Header.BlockHash)
	}
	return nil
}

func (r *Reducer) sendAgreement(hash []byte, svs []*agreement.StepVotes) {
	// first, sign the two StepVotes
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
	if err := r.signer.SendAuthenticated(topics.Agreement, hash, eventBuf, r.ID()); err != nil {
		lg.WithField("category", "BUG").WithError(err).Errorln("error in Ed25519 signing and gossip the agreement")
	}
}

func stepVotesAreValid(svs []*agreement.StepVotes) bool {
	return len(svs) == 2 && svs[0] != nil && svs[1] != nil
}
