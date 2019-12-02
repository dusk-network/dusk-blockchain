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
	eventPlayer  consensus.EventPlayer
	handler      *handler
	accumulator  *Accumulator
	keys         key.ConsensusKeys
	workerAmount int
	quitChan     chan struct{}

	agreementID uint32
	round       uint64
}

// newComponent is used by the agreement factory to instantiate the component
func newComponent(publisher eventbus.Publisher, keys key.ConsensusKeys, workerAmount int) *agreement {
	return &agreement{
		publisher:    publisher,
		keys:         keys,
		workerAmount: workerAmount,
		quitChan:     make(chan struct{}, 1),
	}
}

// Initialize the agreement component, by creating the handler and the accumulator, and
// return a listener for Agreement messages.
// Implements consensus.Component.
func (a *agreement) Initialize(eventPlayer consensus.EventPlayer, signer consensus.Signer, r consensus.RoundUpdate) []consensus.TopicListener {
	a.eventPlayer = eventPlayer
	a.handler = newHandler(a.keys, r.P)
	a.accumulator = newAccumulator(a.handler, a.workerAmount)
	a.round = r.Round
	agreementSubscriber := consensus.TopicListener{
		Preprocessors: []eventbus.Preprocessor{consensus.NewRepublisher(a.publisher, topics.Agreement), &consensus.Validator{}},
		Topic:         topics.Agreement,
		Listener:      consensus.NewFilteringListener(a.CollectAgreementEvent, a.Filter, consensus.LowPriority, false),
	}
	a.agreementID = agreementSubscriber.Listener.ID()

	go a.listen()
	return []consensus.TopicListener{agreementSubscriber}
}

// Returns the listener ID for the agreement component.
// Implements consensus.Component.
func (a *agreement) ID() uint32 {
	return a.agreementID
}

// Filter an incoming Agreement message, by checking whether it was sent by a valid
// member of the voting committee for the given round and step.
func (a *agreement) Filter(hdr header.Header) bool {
	return !a.handler.IsMember(hdr.PubKeyBLS, hdr.Round, hdr.Step)
}

// CollectAgreementEvent is the callback to get Events from the Coordinator. It forwards
// the events to the accumulator until Quorum is reached
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

// Listen for results coming from the accumulator.
func (a *agreement) listen() {
	select {
	case evs := <-a.accumulator.CollectedVotesChan:
		lg.WithField("id", a.agreementID).Debugln("quorum reached")
		cert := a.generateCertificate(evs[0])
		go a.sendFinalize(cert, evs[0].BlockHash)
	case <-a.quitChan:
	}
}

// Send the certificate over to the Coordinator, signalling that the
// round has finished.
func (a *agreement) sendFinalize(cert *block.Certificate, blockHash []byte) {
	buf := new(bytes.Buffer)
	if err := encoding.WriteUint64LE(buf, a.round); err != nil {
		lg.WithField("category", "BUG").WithError(err).Errorln("error marshalling round")
		return
	}

	if err := encoding.Write256(buf, blockHash); err != nil {
		lg.WithField("category", "BUG").WithError(err).Errorln("error marshalling block hash")
		return
	}

	if err := marshalling.MarshalCertificate(buf, cert); err != nil {
		lg.WithField("category", "BUG").WithError(err).Errorln("error marshalling certificate")
		return
	}

	a.publisher.Publish(topics.Finalize, buf)
}

// Finalize the agreement component, by pausing event streaming, and shutting down
// the accumulator. Additionally, it ensures the `listen` goroutine is shut down.
// The agreement component is no longer usable after this method call.
// Implements consensus.Component.
func (a *agreement) Finalize() {
	a.eventPlayer.Pause(a.agreementID)
	a.accumulator.Stop()
	select {
	case a.quitChan <- struct{}{}:
	default:
	}
}

// Generate a block certificate from an agreement message.
func (a *agreement) generateCertificate(ag Agreement) *block.Certificate {
	return &block.Certificate{
		StepOneBatchedSig: ag.VotesPerStep[0].Signature.Compress(),
		StepTwoBatchedSig: ag.VotesPerStep[1].Signature.Compress(),
		Step:              ag.Header.Step,
		StepOneCommittee:  ag.VotesPerStep[0].BitSet,
		StepTwoCommittee:  ag.VotesPerStep[1].BitSet,
	}
}
