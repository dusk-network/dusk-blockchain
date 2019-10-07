package reduction

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/msg"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/selection"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

type (
	// scoreCollector is a helper to obtain a score channel already wired to the
	// EventBus and fully functioning.
	scoreCollector struct {
		bestVotedScoreHashChan chan<- *selection.ScoreEvent
	}
)

// initBestScoreUpdate is the utility function to create and wire a channel for
// notifications of the best ScoreEvent.
func initBestScoreUpdate(subscriber eventbus.Subscriber) chan *selection.ScoreEvent {
	bestVotedScoreHashChan := make(chan *selection.ScoreEvent, 1)
	collector := &scoreCollector{
		bestVotedScoreHashChan: bestVotedScoreHashChan,
	}
	eventbus.NewTopicListener(subscriber, collector, string(msg.BestScoreTopic), eventbus.ChannelType)
	return bestVotedScoreHashChan
}

func (sc *scoreCollector) Collect(r bytes.Buffer) error {
	// copy shared pointer
	ev := &selection.ScoreEvent{}
	if err := selection.UnmarshalScoreEvent(&r, ev); err != nil {
		return err
	}
	if len(ev.VoteHash) == 32 {
		sc.bestVotedScoreHashChan <- ev
	} else {
		sc.bestVotedScoreHashChan <- nil
	}

	return nil
}
