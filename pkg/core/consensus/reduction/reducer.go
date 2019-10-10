package reduction

import (
	"bytes"
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	log "github.com/sirupsen/logrus"
)

var emptyHash = [32]byte{}

type reducer struct {
	publisher         eventbus.Publisher
	rpcBus            *rpcbus.RPCBus
	signer            *consensus.Signer
	requestStepUpdate func()

	svs     []agreement.StepVotes
	handler *reductionHandler

	lock sync.RWMutex
	// TODO: rename
	aggregatorSemaphore bool

	aggregator *aggregator
	timer      *timer
}

func NewComponent(publisher eventbus.Publisher, rpcBus *rpcbus.RPCBus, keys user.Keys) *reducer {
	return &reducer{
		publisher: publisher,
		rpcBus:    rpcBus,
		svs:       make([]agreement.StepVotes, 0, 2),
	}
}

func (r *reducer) Initialize(provisioners user.Provisioners) []consensus.Subscribers {
	r.handler = newReductionHandler(provisioners, keys)
	reductionSubscriber := consensus.Subscriber{
		topic:    topics.Reduction,
		listener: consensus.NewFilteringListener(r.CollectReductionEvent, r.Filter),
	}

	scoreSubscriber := consensus.Subscriber{
		topic:    topics.BestScore,
		listener: consensus.NewSimpleListener(r.CollectBestScore),
	}

	return []Subscriber{reductionSubscriber, scoreSubscriber}
}

func (r *reducer) Finalize() {
	r.timer.Stop()
}

func (r *reducer) resetStepVotes() {
	r.svs = make([]agreement.StepVotes, 0, 2)
}

func (r *reducer) addStepVotes(sv agreement.StepVotes, blockHash []byte) {
	r.svs = append(r.svs, sv)

	if r.isAggregatorActive() {
		if err := r.verifyCandidateBlock(blockHash); err != nil {
			blockHash = emptyHash
		}
	}

	r.collectStepVotes(r.svs, blockHash)
	r.requestStepUpdate()

	if !r.isAggregatorActive() {
		r.publishRegeneration()
	}
}

func (r *reducer) collectStepVotes(svs []agreement.StepVotes, blockHash []byte) {
	switch len(svs) {
	case 1:
		r.sendReductionVote(blockHash)
	case 2:
		sig, err := r.signer.BLSSign(svs)
		if err != nil {
		}

		ag, err := r.generateAgreement(svs, sig)
		if err != nil {
		}

		r.sendAgreement(ag)
		r.resetStepVotes()
	default:
		panic("stepvotes slice is of unexpected length")
	}
}

func (r *reducer) CollectBestScore(blockHash []byte) {
	r.activateAggregator()
	r.sendReductionVote(blockHash)
	r.startReduction()
}

func (r *reducer) CollectReductionEvent(m bytes.Buffer, hdr header.Header) error {
	ev := New()
	if err := reduction.Unmarshal(&m, ev); err != nil {
		return err
	}

	if err := r.handler.VerifySignature(hdr, ev.SignedHash); err != nil {
		return err
	}

	r.aggregator.collectVote(*ev, hdr)
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
	r.aggregator = newAggregator()
	r.aggregator.Start()
}

func (r *reducer) verifyCandidateBlock(blockHash []byte) error {
	// If our result was not a zero value hash, we should first verify it
	// before voting on it again
	if !bytes.Equal(blockHash, emptyHash[:]) {
		req := rpcbus.NewRequest(*(bytes.NewBuffer(hash)), 5)
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

func (r *reducer) publishRegeneration() {
	r.publisher.Publish(topics.BlockRegeneration, bytes.Buffer{})
}

func (r *reducer) sendReductionVote(hash []byte) {
	ev, err := r.generateReduction(blockHash)
	if err != nil {
	}

	r.sendReduction(ev)
}

func (r *reducer) generateReduction(hash []byte) (*bytes.Buffer, error) {
	sig, err := r.signer.BLSSign(hash)
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

func (r *reducer) sendReduction(m *bytes.Buffer) {
	r.publisher.Publish(topics.Reduction, m)
}

func (r *reducer) generateAgreement(svs []agreement.StepVotes, sig []byte) (*bytes.Buffer, error) {
	ev := agreement.New()
	ev.SignedVotes = sig
	ev.VotesPerStep = svs

	buf := new(bytes.Buffer)
	if err := agreement.Marshal(buf, ev); err != nil {
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
