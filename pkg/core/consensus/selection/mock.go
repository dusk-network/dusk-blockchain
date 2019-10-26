package selection

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	crypto "github.com/dusk-network/dusk-crypto/hash"
)

// MockSelectionEventBuffer mocks a Selection event, marshals it, and returns the
// resulting buffer.
func MockSelectionEventBuffer(hash []byte, bidList user.BidList) *bytes.Buffer {
	se := MockSelectionEvent(hash, bidList)
	r := new(bytes.Buffer)
	_ = MarshalScoreEvent(r, se)
	return r
}

// MockSelectionEvent mocks a Selection event and returns it.
func MockSelectionEvent(hash []byte, bidList user.BidList) *ScoreEvent {
	score, _ := crypto.RandEntropy(32)
	proof, _ := crypto.RandEntropy(1477)
	z, _ := crypto.RandEntropy(32)
	subset := bidList.Subset(len(bidList))
	seed, _ := crypto.RandEntropy(33)

	return &ScoreEvent{
		Score:         score,
		Proof:         proof,
		Z:             z,
		Seed:          seed,
		BidListSubset: bidListToBytes(subset),
		PrevHash:      hash,
		VoteHash:      hash,
	}
}

func bidListToBytes(bidList []user.Bid) []byte {
	buf := new(bytes.Buffer)
	for _, bid := range bidList {
		if err := encoding.Write256(buf, bid.X[:]); err != nil {
			panic(err)
		}
	}

	return buf.Bytes()
}
