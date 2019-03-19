package voting

import (
	"bytes"
	"encoding/hex"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// Voter is responsible for signing and sending out consensus related
// voting messages to the wire.
type Voter struct {
	eventBus                          *wire.EventBus
	outgoingBlockReductionSubscriber  *wire.EventSubscriber
	outgoingSigSetReductionSubscriber *wire.EventSubscriber
	outgoingSigSetSubScriber          *wire.EventSubscriber
	outgoingBlockAgreementSubscriber  *wire.EventSubscriber
	outgoingSigSetAgreementSubscriber *wire.EventSubscriber

	blockReductionChannel  chan *reduction
	sigSetReductionChannel chan *sigSetReduction

	*user.Keys
	committee user.Committee
}

// NewVoter will return an initialized Voter struct.
func NewVoter(eventBus *wire.EventBus, keys *user.Keys,
	committee user.Committee) *Voter {

	blockReductionChannel := make(chan *reduction, 2)
	sigSetReductionChannel := make(chan *sigSetReduction, 2)

	blockReductionCollector := &blockReductionCollector{blockReductionChannel}
	blockReductionSubscriber := wire.NewEventSubscriber(eventBus, blockReductionCollector,
		string(msg.OutgoingReductionTopic))

	sigSetReductionCollector := &sigSetReductionCollector{sigSetReductionChannel}
	sigSetReductionSubscriber := wire.NewEventSubscriber(eventBus, sigSetReductionCollector,
		string(msg.OutgoingReductionTopic))

	return &Voter{
		eventBus:                          eventBus,
		outgoingBlockReductionSubscriber:  blockReductionSubscriber,
		outgoingSigSetReductionSubscriber: sigSetReductionSubscriber,
		blockReductionChannel:             blockReductionChannel,
		sigSetReductionChannel:            sigSetReductionChannel,
		Keys:                              keys,
		committee:                         committee,
	}
}

func (v Voter) addPubKeyAndSig(message *bytes.Buffer,
	signature []byte) (*bytes.Buffer, error) {

	buffer := bytes.NewBuffer([]byte(*v.EdPubKey))
	if err := encoding.Write512(buffer, signature); err != nil {
		return nil, err
	}

	if _, err := buffer.Write(message.Bytes()); err != nil {
		return nil, err
	}

	return buffer, nil
}

func (v Voter) checkEligibility(round uint64, step uint8) (bool, error) {
	votingCommittee, err := v.committee.GetVotingCommittee(round, step)
	if err != nil {
		return false, err
	}

	pubKeyStr := hex.EncodeToString(v.BLSPubKey.Marshal())
	return votingCommittee[pubKeyStr] > 0, nil
}

// Listen will set the Voter up to listen for incoming requests to vote.
func (v Voter) Listen() {
	go v.outgoingBlockAgreementSubscriber.Accept()
	go v.outgoingBlockReductionSubscriber.Accept()
	go v.outgoingSigSetAgreementSubscriber.Accept()
	go v.outgoingSigSetReductionSubscriber.Accept()
	go v.outgoingSigSetSubScriber.Accept()

	for {
		select {
		case br := <-v.blockReductionChannel:
			eligible, err := v.checkEligibility(br.round, br.step)
			if err != nil {
				// Log
				return
			}

			if eligible {
				if err := v.voteBlockReduction(br); err != nil {
					// Log
					return
				}
			}
		case sr := <-v.sigSetReductionChannel:
			eligible, err := v.checkEligibility(sr.round, sr.step)
			if err != nil {
				// Log
				return
			}

			if eligible {
				if err := v.voteSigSetReduction(sr); err != nil {
					// Log
					return
				}
			}
		}
	}
}
