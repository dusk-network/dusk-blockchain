// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package reduction

import (
	"fmt"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/util"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/sortedset"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

var lg = log.WithField("process", "consensus").WithField("actor", "aggr")

// The Aggregator acts as a de facto storage unit for Reduction messages. Any message
// it receives will be Aggregated into a StepVotes struct, organized by block hash.
// Once the key set for a StepVotes of a certain block hash reaches quorum, this
// StepVotes is passed on to the Reducer by use of the `haltChan` channel.
// An Aggregator should be instantiated on a per-step basis and is no longer usable
// after reaching quorum and sending on `haltChan`.
type Aggregator struct {
	handler *Handler

	voteSets map[string]struct {
		*message.StepVotes
		sortedset.Cluster
	}
}

// NewAggregator returns an instantiated Aggregator, ready for use by both
// reduction steps.
func NewAggregator(handler *Handler) *Aggregator {
	return &Aggregator{
		handler: handler,
		voteSets: make(map[string]struct {
			*message.StepVotes
			sortedset.Cluster
		}),
	}
}

// CollectVote collects a Reduction message, and add its sender public key and signature to the
// StepVotes/Set kept under the corresponding block hash. If the Set reaches or exceeds
// quorum, a result is created with the voted hash and the related StepVotes
// added. The validation of the candidate block is left to the caller.
func (a *Aggregator) CollectVote(ev message.Reduction) *Result {
	hdr := ev.State()
	hash := string(hdr.BlockHash)
	sv, found := a.voteSets[hash]

	if !found {
		sv.StepVotes = message.NewStepVotes()
		sv.Cluster = sortedset.NewCluster()
	}

	// Each committee has 64 slots. If a Provisioner is extracted into
	// multiple slots, then he/she only needs to send one vote which can be
	// taken account as a vote for all his/her slots. Otherwise, if a
	// Provisioner is only extracted to one slot per committee, then a single
	// vote is taken into account (if more votes for the same slot are
	// propagated, those are discarded).

	if sv.Cluster.Contains(hdr.PubKeyBLS) {
		log.Warn("Disacrding duplicated votes from a Provisioner")
		return nil
	}

	// Aggregated Signatures
	if err := sv.StepVotes.Add(ev.SignedHash); err != nil {
		// adding the vote to the cluster failed. This is a programming error
		panic(err)
	}

	votes := a.handler.VotesFor(hdr.PubKeyBLS, hdr.Round, hdr.Step)
	for i := 0; i < votes; i++ {
		sv.Cluster.Insert(hdr.PubKeyBLS)
	}

	a.voteSets[hash] = sv
	total := sv.Cluster.TotalOccurrences()
	roundQuorumTarget := a.handler.Quorum(hdr.Round)

	if log.GetLevel() >= logrus.DebugLevel {
		c := a.handler.Committee(hdr.Round, hdr.Step)

		log := consensus.WithFields(hdr.Round, hdr.Step, "vote_received",
			hdr.BlockHash, nil, &c, &sv.Cluster, &a.handler.Provisioners)

		log.WithField("total", total).
			WithField("vote_from", util.StringifyBytes(hdr.PubKeyBLS)).
			WithField("num", votes).
			WithField("quorum_target", roundQuorumTarget).Debug()
	}

	if total >= roundQuorumTarget {
		// quorum reached
		a.addBitSet(sv.StepVotes, sv.Cluster, hdr.Round, hdr.Step)

		if logrus.GetLevel() >= logrus.DebugLevel {
			c := a.handler.Committee(hdr.Round, hdr.Step)
			log := consensus.WithFields(hdr.Round, hdr.Step, "quorum_reached",
				hdr.BlockHash, nil, &c, &sv.Cluster, &a.handler.Provisioners)

			log.
				WithField("total_votes", total).
				WithField("quorum_target", roundQuorumTarget).
				WithField("bitset", fmt.Sprintf("%b", sv.BitSet)).Debug("")
		}

		return &Result{hdr.BlockHash, *sv.StepVotes}
	}

	// quorum not reached
	return nil
}

func (a *Aggregator) addBitSet(sv *message.StepVotes, cluster sortedset.Cluster, round uint64, step uint8) {
	committee := a.handler.Committee(round, step)
	sv.BitSet = committee.Bits(cluster.Set)
}

// Log dumps current state of voteSets in passed logrus.
func (a *Aggregator) Log(l *logrus.Entry, round uint64, step uint8) {
	target := a.handler.Quorum(round)

	for hash, sv := range a.voteSets {
		total := sv.Cluster.TotalOccurrences()

		l.WithField("hash", util.StringifyBytes([]byte(hash))).
			WithField("total", total).
			WithField("round", round).
			WithField("step", step).
			WithField("quorum_target", target).Info()
	}
}
