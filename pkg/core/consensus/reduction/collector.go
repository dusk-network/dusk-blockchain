package reduction

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/selection"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

type (
	//scoreCollector is a helper to obtain a score channel already wired to the EventBus and fully functioning
	scoreCollector struct {
		bestVotedScoreHashChan chan<- []byte
	}
)

// initBestScoreUpdate is the utility function to create and wire a channel for notifications of the best ScoreEvent
func initBestScoreUpdate(subscriber wire.EventSubscriber) chan []byte {
	bestVotedScoreHashChan := make(chan []byte, 1)
	collector := &scoreCollector{
		bestVotedScoreHashChan: bestVotedScoreHashChan,
	}
	go wire.NewTopicListener(subscriber, collector, string(msg.BestScoreTopic)).Accept()
	return bestVotedScoreHashChan
}

func (sc *scoreCollector) Collect(r *bytes.Buffer) error {
	ev := &selection.ScoreEvent{}
	if err := selection.UnmarshalScoreEvent(r, ev); err != nil {
		return err
	}
	if len(ev.VoteHash) == 32 {
		sc.bestVotedScoreHashChan <- ev.VoteHash
	} else {
		sc.bestVotedScoreHashChan <- make([]byte, 32)
	}
	return nil
}
