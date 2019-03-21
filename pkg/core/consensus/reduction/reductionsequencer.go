package reduction

import (
	"bytes"
	"encoding/hex"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
)

type reductionSequencer struct {
	outgoingReductionChannel chan []byte
	outgoingAgreementChannel chan []byte
	voteSetStore             []*msg.Vote

	incomingReductionChannel chan *Event
	quorum                   int
	timeOut                  time.Duration
}

func newReductionSequencer(quorum int, timeOut time.Duration) *reductionSequencer {
	return &reductionSequencer{
		outgoingReductionChannel: make(chan []byte, 1),
		outgoingAgreementChannel: make(chan []byte, 1),
		incomingReductionChannel: make(chan *Event, 100),
		quorum:                   quorum,
		timeOut:                  timeOut,
	}
}

func (rs *reductionSequencer) startSequence() {
	hash1 := rs.selectHash()
	rs.dispatchReductionVote(hash1)
	hash2 := rs.selectHash()

	// if we got the desired results, then we send out an agreement vote
	if rs.sequenceSuccessful(hash1, hash2) {
		rs.dispatchAgreementVote(hash2)
	}
}

func (rs reductionSequencer) sequenceSuccessful(hash1, hash2 []byte) bool {
	notNil := hash1 != nil && hash2 != nil
	sameResult := bytes.Equal(hash1, hash2)
	correctVoteSetLength := len(rs.voteSetStore) >= rs.quorum

	return notNil && sameResult && correctVoteSetLength
}

func (rs reductionSequencer) dispatchReductionVote(hash []byte) {
	rs.outgoingReductionChannel <- hash
}

func (rs reductionSequencer) dispatchAgreementVote(hash []byte) error {
	voteSetBytes, err := msg.EncodeVoteSet(rs.voteSetStore)
	if err != nil {
		return err
	}

	hashAndVoteSet := append(hash, voteSetBytes...)
	rs.outgoingAgreementChannel <- hashAndVoteSet
	return nil
}

func (rs *reductionSequencer) selectHash() []byte {
	// Keep track of how many votes have been cast for a specific hash
	votesPerHash := make(map[string]uint8)

	timer := time.NewTimer(rs.timeOut)

	for {
		select {
		case m := <-rs.incomingReductionChannel:
			// Add vote for this block
			votedHashStr := hex.EncodeToString(m.VotedHash)
			votesPerHash[votedHashStr]++

			// Add vote to vote set
			vote := createVote(m)
			rs.voteSetStore = append(rs.voteSetStore, vote)

			if int(votesPerHash[votedHashStr]) >= rs.quorum {
				// Clean up the vote set, to remove votes for other blocks.
				for i, vote := range rs.voteSetStore {
					if !bytes.Equal(vote.VotedHash, m.VotedHash) {
						rs.voteSetStore = append(rs.voteSetStore[:i],
							rs.voteSetStore[i+1:]...)
					}
				}

				return m.VotedHash
			}
		case <-timer.C:
			// Return empty hash if we did not get a winner before timeout
			return make([]byte, 32)
		}
	}
}

func createVote(m *Event) *msg.Vote {
	return &msg.Vote{
		VotedHash:  m.VotedHash,
		PubKeyBLS:  m.PubKeyBLS,
		SignedHash: m.SignedHash,
		Step:       m.Step,
	}
}
