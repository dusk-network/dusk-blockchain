package reduction

import (
	"bytes"
	"sync"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	log "github.com/sirupsen/logrus"
)

var _ consensus.Component = (*reducer)(nil)

var emptyHash = [32]byte{}

type reducer struct {
	publisher eventbus.Publisher
	rpcBus    *rpcbus.RPCBus
	keys      user.Keys
	store     consensus.Store

	svs     []agreement.StepVotes
	handler *reductionHandler

	lock sync.RWMutex
	// TODO: rename
	aggregatorSemaphore bool

	aggregator *aggregator
	timeOut    time.Duration
	timer      *timer
}

// NewComponent returns an uninitialized reduction component.
func newComponent(publisher eventbus.Publisher, rpcBus *rpcbus.RPCBus, keys user.Keys, timeOut time.Duration) *reducer {
	return &reducer{
		publisher: publisher,
		rpcBus:    rpcBus,
		svs:       make([]agreement.StepVotes, 0, 2),
		keys:      keys,
		timeOut:   timeOut,
	}
}

// Initialize the reduction component, by instantiating the handler and creating
// the topic subscribers.
// Implements consensus.Component
func (r *reducer) Initialize(store consensus.Store, ru consensus.RoundUpdate) []consensus.Subscriber {
	r.store = store
	r.handler = newReductionHandler(r.keys, ru.P)
	r.timer = &timer{r: r}

	reductionSubscriber := consensus.Subscriber{
		Topic:    topics.Reduction,
		Listener: consensus.NewFilteringListener(r.CollectReductionEvent, r.Filter),
	}

	scoreSubscriber := consensus.Subscriber{
		Topic:    topics.BestScore,
		Listener: consensus.NewSimpleListener(r.CollectBestScore),
	}

	return []consensus.Subscriber{reductionSubscriber, scoreSubscriber}
}

// Finalize the reducer component by killing the timer, if it is still running.
// This will stop a reduction cycle short, and renders this reducer useless
// after calling.
func (r *reducer) Finalize() {
	r.timer.stop()
}

func (r *reducer) resetStepVotes() {
	r.svs = make([]agreement.StepVotes, 0, 2)
}

func (r *reducer) addStepVotes(sv agreement.StepVotes, blockHash []byte) {
	r.svs = append(r.svs, sv)

	if r.isAggregatorActive() {
		if err := r.verifyCandidateBlock(blockHash); err != nil {
			blockHash = emptyHash[:]
		}
	}

	r.collectStepVotes(r.svs, blockHash)
	r.store.RequestStepUpdate()

	if !r.isAggregatorActive() {
		r.publishRegeneration()
	}
}

func (r *reducer) collectStepVotes(svs []agreement.StepVotes, blockHash []byte) {
	switch len(svs) {
	case 1:
		r.sendReductionVote(blockHash)
	case 2:
		m := new(bytes.Buffer)
		if err := agreement.MarshalVotes(m, svs); err != nil {
			//TODO: handle
		}
		sig, err := r.store.RequestSignature(m.Bytes())
		if err != nil {
			//TODO: handle
		}

		ag, err := r.generateAgreement(blockHash, svs, sig)
		if err != nil {
			//TODO: handle
		}

		r.sendAgreement(ag)
		r.resetStepVotes()
	default:
		// This function should never be called with any other length than 1 or 2.
		panic("stepvotes slice is of unexpected length")
	}
}

// CollectBestScore activates the 2-step reduction cycle.
func (r *reducer) CollectBestScore(e consensus.Event) error {
	r.activateAggregator()
	r.sendReductionVote(e.Payload.Bytes())
	r.startReduction()
	return nil
}

func (r *reducer) CollectReductionEvent(e consensus.Event) error {
	ev := New()
	if err := Unmarshal(&e.Payload, ev); err != nil {
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

func (r *reducer) SetStep(step uint8) {
	r.timer.stop()

	if r.isAggregatorActive() {
		r.startReduction()
		r.deferAggregatorDeactivation()
	}
}

func (r *reducer) startReduction() {
	r.timer.start(r.timeOut)
	r.aggregator = newAggregator(r)
	//r.aggregator.Start()
}

func (r *reducer) verifyCandidateBlock(blockHash []byte) error {
	// If our result was not a zero value hash, we should first verify it
	// before voting on it again
	if !bytes.Equal(blockHash, emptyHash[:]) {
		req := rpcbus.NewRequest(*(bytes.NewBuffer(blockHash)), 5)
		if _, err := r.rpcBus.Call(rpcbus.VerifyCandidateBlock, req); err != nil {
			log.WithFields(log.Fields{
				"process": "reduction",
				"error":   err,
			}).Errorln("verifying the candidate block failed")
			return err
		}
	}

	return nil
}

var regenerationPackage = new(bytes.Buffer)

func (r *reducer) publishRegeneration() {
	r.publisher.Publish(topics.BlockRegeneration, regenerationPackage)
}

func (r *reducer) sendReductionVote(hash []byte) error {
	ev, err := r.generateReduction(hash)
	if err != nil {
		return err
	}

	return r.sendReduction(hash, ev)
}

func (r *reducer) generateReduction(hash []byte) (*bytes.Buffer, error) {

	sig, err := r.store.RequestSignature(hash)
	if err != nil {
		return nil, err
	}

	vote := new(bytes.Buffer)
	if err := encoding.Write256(vote, hash); err != nil {
		return nil, err
	}

	if err := encoding.WriteBLS(vote, sig); err != nil {
		return nil, err
	}

	return vote, nil
}

func (r *reducer) sendReduction(hash []byte, m *bytes.Buffer) error {
	if err := r.store.WriteHeader(hash, m); err != nil {
		return err
	}
	r.publisher.Publish(topics.Reduction, m)
	return nil
}

func (r *reducer) generateAgreement(hash []byte, svs []agreement.StepVotes, sig []byte) (*bytes.Buffer, error) {
	ev := agreement.Agreement{}
	ev.SignedVotes = sig
	ev.VotesPerStep = svs

	buf := new(bytes.Buffer)
	if err := agreement.Marshal(buf, ev); err != nil {
		return nil, err
	}

	if err := r.store.WriteHeader(hash, buf); err != nil {
		return nil, err
	}

	return buf, nil
}

func (r *reducer) sendAgreement(m *bytes.Buffer) {
	r.publisher.Publish(topics.Agreement, m)
}

func (r *reducer) isAggregatorActive() bool {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.aggregatorSemaphore == true
}

func (r *reducer) deferAggregatorDeactivation() {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.aggregatorSemaphore = false
}

func (r *reducer) activateAggregator() {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.aggregatorSemaphore = true
}
