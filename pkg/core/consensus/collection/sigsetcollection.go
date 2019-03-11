package collection

import (
	"encoding/hex"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
)

type SetCollector struct {
	Round uint64
	Step  uint8
	Queue *user.Queue
}

// SelectBestSignatureSet will receive signature set candidate messages through
// setCandidateChannel for a set amount of time. The function will store the
// vote set of the node with the highest stake. When the timer runs out,
// the stored vote set is returned.
func SelectBestSignatureSet(timerLength time.Duration, nodeStakes map[string]uint64,
	setCandidateChannel chan *payload.MsgConsensus) (bestVoteSet []*payload.Vote) {

	// Variable to keep track of the highest score we've received.
	var highest uint64

	timer := time.NewTimer(timerLength)

	for {
		select {
		case <-timer.C:
			return
		case m := <-setCandidateChannel:
			// Retrieve payload
			pl := m.Payload.(*payload.SigSetCandidate)

			// Retrieve the sender node's stake
			pubKeyStr := hex.EncodeToString(m.PubKey)
			stake := nodeStakes[pubKeyStr]

			if stake > highest {
				highest = stake
				bestVoteSet = pl.SignatureSet
			}
		}
	}
}
