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
	stepper   consensus.Stepper
	signer    consensus.Signer

	svs     []*agreement.StepVotes
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
		svs:       make([]*agreement.StepVotes, 0, 2),
		keys:      keys,
		timeOut:   timeOut,
	}
}

// Initialize the reduction component, by instantiating the handler and creating
// the topic subscribers.
// Implements consensus.Component
func (r *reducer) Initialize(stepper consensus.Stepper, signer consensus.Signer, ru consensus.RoundUpdate) []consensus.Subscriber {
	r.stepper = stepper
	r.signer = signer
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
	r.svs = make([]*agreement.StepVotes, 0, 2)
}

func (r *reducer) addStepVotes(sv *agreement.StepVotes, blockHash []byte) {
	r.svs = append(r.svs, sv)

	if r.isAggregatorActive() {
		if err := r.verifyCandidateBlock(blockHash); err != nil {
			blockHash = emptyHash[:]
		}
	}

	r.collectStepVotes(r.svs, blockHash)
	r.stepper.RequestStepUpdate()

	if !r.isAggregatorActive() {
		r.publishRegeneration()
	}
}

func (r *reducer) collectStepVotes(svs []*agreement.StepVotes, blockHash []byte) {
	switch len(svs) {
	case 1:
		if err := r.sendReduction(blockHash); err != nil {
			//TODO: handle
		}
	case 2:
		if err := r.sendAgreement(blockHash, svs); err != nil {
			//TODO: handle
		}

		r.resetStepVotes()
	default:
		// This function should never be called with any other length than 1 or 2.
		panic("stepvotes slice is of unexpected length")
	}
}

// CollectBestScore activates the 2-step reduction cycle.
func (r *reducer) CollectBestScore(e consensus.Event) error {
	r.activateAggregator()
	r.sendReduction(e.Payload.Bytes())
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

func (r *reducer) sendAgreement(hash []byte, svs []*agreement.StepVotes) error {
	// first we sign the marshalled StepVotes
	payloadBuf := new(bytes.Buffer)
	if err := agreement.MarshalVotes(payloadBuf, svs); err != nil {
		//TODO: handle
	}

	sig, err := r.signer.Sign(hash, payloadBuf.Bytes())
	if err != nil {
		//TODO: handle
	}

	// then we create the full BLS signed Agreement
	ev := agreement.Agreement{}
	ev.SetSignature(sig)
	ev.VotesPerStep = svs

	eventBuf := new(bytes.Buffer)
	if err := agreement.Marshal(eventBuf, ev); err != nil {
		return err
	}

	// then we forward the marshalled Agreement to the store to be sent
	return r.signer.SendAuthenticated(topics.Agreement, hash, eventBuf)
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
