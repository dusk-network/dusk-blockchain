package generation

import (
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
)

type state struct {
	roundChan        <-chan consensus.RoundUpdate
	regenerationChan <-chan consensus.AsyncState
	*broker

	currentState consensus.RoundUpdate
}

func NewState(eventBroker wire.EventBroker) *state {
	return &state{
		roundChan:        consensus.InitRoundUpdate(eventBroker),
		regenerationChan: consensus.InitBlockRegenerationCollector(eventBroker),
	}
}

func (s *state) Wire(broker *broker) {
	s.broker = broker
	go broker.Listen()
	go s.listen()
}

func (s *state) listen() {
	for {
		select {
		case roundUpdate := <-s.roundChan:
			s.currentState = roundUpdate
			s.broker.Generate(roundUpdate)
		case state := <-s.regenerationChan:
			if state.Round == s.currentState.Round {
				s.broker.Generate(s.currentState)
			}
		}
	}
}
