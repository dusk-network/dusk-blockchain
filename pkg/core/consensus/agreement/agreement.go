package agreement

import (
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-wallet/v2/key"
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
	a.handler = NewHandler(a.keys, r.P)
	a.accumulator = newAccumulator(a.handler, a.workerAmount)
	a.round = r.Round
	agreementSubscriber := consensus.TopicListener{
		Topic:    topics.Agreement,
		Listener: consensus.NewFilteringListener(a.CollectAgreementEvent, a.Filter, consensus.LowPriority, false),
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
func (a *agreement) CollectAgreementEvent(packet consensus.InternalPacket) error {
	// casting to Agreement
	aggro := packet.(message.Agreement)

	lg.WithFields(log.Fields{
		"agreement": aggro,
		"id":        a.agreementID,
	}).Debugln("received event")
	a.accumulator.Process(aggro)
	return nil
}

// Listen for results coming from the accumulator.
func (a *agreement) listen() {
	select {
	case evs := <-a.accumulator.CollectedVotesChan:
		lg.WithField("id", a.agreementID).Debugln("quorum reached")
		// Start a goroutine here to release the lock held by
		// Coordinator.CollectEvent
		// Send the Agreement to the Certificate Collector within the Chain
		go a.sendCertificate(evs[0])
	case <-a.quitChan:
	}
}

func (a *agreement) sendCertificate(ag message.Agreement) {
	msg := message.New(topics.Agreement, ag)
	a.publisher.Publish(topics.Certificate, msg)
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
