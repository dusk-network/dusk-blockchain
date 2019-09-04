package reduction

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/msg"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/selection"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
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
func initBestScoreUpdate(subscriber wire.EventSubscriber) chan *selection.ScoreEvent {
	bestVotedScoreHashChan := make(chan *selection.ScoreEvent, 1)
	collector := &scoreCollector{
		bestVotedScoreHashChan: bestVotedScoreHashChan,
	}
	go wire.NewTopicListener(subscriber, collector, string(msg.BestScoreTopic)).Accept()
	return bestVotedScoreHashChan
}

func (sc *scoreCollector) Collect(r *bytes.Buffer) error {
	// copy shared pointer
	copyBuf := *r
	ev := &selection.ScoreEvent{Certificate: block.EmptyCertificate()}
	if err := selection.UnmarshalScoreEvent(&copyBuf, ev); err != nil {
		return err
	}
	if len(ev.VoteHash) == 32 {
		sc.bestVotedScoreHashChan <- ev
	} else {
		sc.bestVotedScoreHashChan <- nil
	}

	return nil
}
