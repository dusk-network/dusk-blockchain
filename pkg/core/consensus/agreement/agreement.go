package agreement

import (
	"bytes"
	"encoding/hex"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/marshalling"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-wallet/block"
	"github.com/dusk-network/dusk-wallet/key"
	log "github.com/sirupsen/logrus"
)

var lg = log.WithField("process", "agreement")

var _ consensus.Component = (*agreement)(nil)

type agreement struct {
	publisher    eventbus.Publisher
	handler      *handler
	accumulator  *Accumulator
	keys         key.ConsensusKeys
	workerAmount int

	agreementID uint32
	round       uint64
}

// newComponent is used by the agreement factory to instantiate the component
func newComponent(publisher eventbus.Publisher, keys key.ConsensusKeys, workerAmount int) *agreement {
	return &agreement{
		publisher:    publisher,
		keys:         keys,
		workerAmount: workerAmount,
	}
}

func (a *agreement) Initialize(eventPlayer consensus.EventPlayer, signer consensus.Signer, r consensus.RoundUpdate) []consensus.TopicListener {
	a.handler = newHandler(a.keys, r.P)
	a.accumulator = newAccumulator(a.handler, a.workerAmount)
	a.round = r.Round
	agreementSubscriber := consensus.TopicListener{
		Preprocessors: []eventbus.Preprocessor{consensus.NewRepublisher(a.publisher, topics.Agreement), &consensus.Validator{}},
		Topic:         topics.Agreement,
		Listener:      consensus.NewFilteringListener(a.CollectAgreementEvent, a.Filter, consensus.LowPriority),
	}
	a.agreementID = agreementSubscriber.Listener.ID()

	go a.listen()
	return []consensus.TopicListener{agreementSubscriber}
}

func (a *agreement) ID() uint32 {
	return a.agreementID
}

func (a *agreement) Filter(hdr header.Header) bool {
	return !a.handler.IsMember(hdr.PubKeyBLS, hdr.Round, hdr.Step)
}

// CollectAgreementEvent is the callback to get Events from the Coordinator. It forwards the events to the accumulator until Quorum is reached
func (a *agreement) CollectAgreementEvent(event consensus.Event) error {
	ev, err := convertToAgreement(event)
	if err != nil {
		return err
	}

	lg.WithFields(log.Fields{
		"round":  event.Header.Round,
		"step":   event.Header.Step,
		"sender": hex.EncodeToString(event.Header.Sender()),
		"id":     a.agreementID,
	}).Debugln("received event")
	a.accumulator.Process(*ev)
	return nil
}

func convertToAgreement(event consensus.Event) (*Agreement, error) {
	ev := New(event.Header)
	if err := Unmarshal(&event.Payload, ev); err != nil {
		return nil, err
	}
	return ev, nil
}

// Listen for results coming from the accumulator
func (a *agreement) listen() {
	evs := <-a.accumulator.CollectedVotesChan
	lg.WithField("id", a.agreementID).Debugln("quorum reached")
	a.sendFinalize()
	a.sendCertificate(evs[0])
}

func (a *agreement) sendCertificate(ag Agreement) {
	cert := a.generateCertificate(ag)
	buf := new(bytes.Buffer)
	if err := encoding.Write256(buf, ag.Header.BlockHash); err != nil {
		lg.WithField("category", "BUG").WithError(err).Errorln("error marshalling block hash")
	}

	if err := marshalling.MarshalCertificate(buf, cert); err != nil {
		lg.WithField("category", "BUG").WithError(err).Errorln("error marshalling certificate")
	}

	a.publisher.Publish(topics.Certificate, buf)
}

func (a *agreement) sendFinalize() {
	buf := new(bytes.Buffer)
	if err := encoding.WriteUint64LE(buf, a.round); err != nil {
		lg.WithError(err).Errorln("what the fuck")
	}

	a.publisher.Publish(topics.Finalize, buf)
}

func (a *agreement) Finalize() {
	a.accumulator.Stop()
}

func (a *agreement) generateCertificate(ag Agreement) *block.Certificate {
	return &block.Certificate{
		StepOneBatchedSig: ag.VotesPerStep[0].Signature.Compress(),
		StepTwoBatchedSig: ag.VotesPerStep[1].Signature.Compress(),
		Step:              ag.Header.Step,
		StepOneCommittee:  ag.VotesPerStep[0].BitSet,
		StepTwoCommittee:  ag.VotesPerStep[1].BitSet,
	}
}
