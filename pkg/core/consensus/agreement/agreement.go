package agreement

import (
	"bytes"
	"sync"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
)

const topic = "agreement"

type Agreement struct {
	eventBus         *wire.EventBus
	agreementChannel <-chan *bytes.Buffer

	round uint64
	step  uint8

	agreeing      bool
	inputChannel  chan *agreementMessage
	outputChannel chan []byte

	queue *user.Queue
}

func NewAgreement(eventBus *wire.EventBus, agreementChannel chan *bytes.Buffer,
	round uint64, step uint8) *Agreement {

	queue := &user.Queue{
		Map: new(sync.Map),
	}

	agreement := &Agreement{
		eventBus:         eventBus,
		agreementChannel: agreementChannel,
		round:            round,
		step:             step,
		queue:            queue,
	}

	agreement.eventBus.Register(topic, agreementChannel)
	return agreement
}

func (a *Agreement) Listen() {
	for {
		// TODO: check queue

		select {
		case messageBytes := <-a.agreementChannel:
			if err := msg.Validate(messageBytes); err != nil {
				break
			}

			message, err := DecodeAgreementMessage(messageBytes)
			if err != nil {
				break
			}

			if a.shouldBeProcessed(message) {
				if !a.agreeing {
					a.agreeing = true
					a.inputChannel = make(chan *agreementMessage, 100)
					a.outputChannel = make(chan []byte, 1)
					go a.selectAgreedHash(a.inputChannel, a.outputChannel)
				}

				a.inputChannel <- message
			} else if a.shouldBeStored(message) {
				a.queue.Put(message.Round, message.Step, message)
			}
		case result := <-a.outputChannel:
			a.agreeing = false
			buffer := bytes.NewBuffer(result)
			a.eventBus.Publish("outgoing", buffer)
		}
	}
}

// selectAgreedHash is a phase-agnostic agreement function. It will receive
// agreement messages from the inputChannel, store the amount of votes it
// has gotten for each step, and once the threshold is reached, the function
// will send the resulting hash into outputChannel, and return.
func (a *Agreement) selectAgreedHash(inputChannel <-chan *agreementMessage,
	outputChannel chan<- []byte) error {

	// Keep track of how many valid votes we have received for any given step.
	stepVotes := make(map[uint32]uint8)

	for {
		m := <-inputChannel

		//

		return nil
	}
}

func (a *Agreement) shouldBeProcessed(m *agreementMessage) bool {
	return m.Round == a.round
}

func (a *Agreement) shouldBeStored(m *agreementMessage) bool {
	return m.Round > a.round
}
