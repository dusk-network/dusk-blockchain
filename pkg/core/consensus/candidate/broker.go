package candidate

import (
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

func LaunchCandidateComponent(eventBroker wire.EventBroker) *broker {
	broker := newBroker(eventBroker)
	go broker.Listen()
	return broker
}

type broker struct {
	filter *consensus.EventFilter
	state  consensus.State
	store  *candidateStore

	roundChan        <-chan uint64
	winningBlockChan <-chan string
}

func newBroker(eventBroker wire.EventBroker) *broker {
	store := newCandidateStore(eventBroker)
	state := consensus.NewState()
	filter := consensus.NewEventFilter(newCandidateHandler(), state, store, false)
	tl := wire.NewTopicListener(eventBroker, filter, string(topics.Candidate))
	go tl.Accept()
	return &broker{
		filter:           filter,
		state:            state,
		store:            store,
		roundChan:        consensus.InitRoundUpdate(eventBroker),
		winningBlockChan: initHashCollector(eventBroker, msg.WinningBlockTopic),
	}
}

func (b *broker) Listen() {
	for {
		select {
		case hash := <-b.winningBlockChan:
			b.store.sendWinningBlock(hash)
		case round := <-b.roundChan:
			b.state.Update(round)
		}
	}
}
