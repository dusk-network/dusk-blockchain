// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package agreement

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/dusk-network/bls12_381-sign/go/cgo/bls"
	"github.com/dusk-network/dusk-blockchain/pkg/config"
	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/committee"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/msg"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/sortedset"
)

// Handler interface is handy for tests.
type Handler interface {
	AmMember(uint64, uint8) bool
	IsMember([]byte, uint64, uint8) bool
	Committee(uint64, uint8) user.VotingCommittee
	Quorum(uint64) int
	VotesFor([]byte, uint64, uint8) int
	Verify(message.Agreement) error
}

type handler struct {
	*committee.Handler
}

// NewHandler returns an initialized handler.
//nolint:golint
func NewHandler(keys key.Keys, p user.Provisioners, seed []byte) *handler {
	return &handler{
		Handler: committee.NewHandler(keys, p, seed),
	}
}

// AmMember checks if we are part of the committee.
func (a *handler) AmMember(round uint64, step uint8) bool {
	return a.Handler.AmMember(round, step, config.ConsensusMaxCommitteeSize)
}

// IsMember delegates the committee.Handler to check if a Provisioner is in the
// committee for a specified round and step.
func (a *handler) IsMember(pubKeyBLS []byte, round uint64, step uint8) bool {
	return a.Handler.IsMember(pubKeyBLS, round, step, config.ConsensusMaxCommitteeSize)
}

// Committee returns a VotingCommittee for a given round and step.
func (a *handler) Committee(round uint64, step uint8) user.VotingCommittee {
	return a.Handler.Committee(round, step, config.ConsensusMaxCommitteeSize)
}

// VotesFor delegates embedded committee.Handler to accumulate a vote for a
// given round.
func (a *handler) VotesFor(pubKeyBLS []byte, round uint64, step uint8) int {
	return a.Handler.VotesFor(pubKeyBLS, round, step, config.ConsensusMaxCommitteeSize)
}

// Quorum returns the amount of committee members necessary to reach a quorum.
func (a *handler) Quorum(round uint64) int {
	return quorum(a.CommitteeSize(round, config.ConsensusMaxCommitteeSize))
}

func quorum(committeeSize int) int {
	return int(math.Ceil(float64(committeeSize) * config.ConsensusQuorumThreshold))
}

// Verify checks the signature of the set.
func (a *handler) Verify(ev message.Agreement) error {
	hdr := ev.State()

	start := time.Now()

	defer func() {
		// Measure duration of a complete verification of an agreement message.
		// Report any duration above 1s.
		alarmThreshold := int64(1000)
		elapsed := time.Since(start)
		ms := elapsed.Milliseconds()

		if ms > alarmThreshold {
			lg.WithField("duration_ms", ms).Info("verify agreement done")
		}
	}()

	if err := verifyWhole(ev); err != nil {
		return fmt.Errorf("failed to verify Agreement Sender: %w", err)
	}

	quorumTarget := a.Quorum(hdr.Round)

	if len(ev.VotesPerStep) != 2 {
		return fmt.Errorf("wrong votesperstep count: %d", len(ev.VotesPerStep))
	}

	for i, votes := range ev.VotesPerStep {
		// the beginning step is the same of the second reduction. Since the
		// consensus steps start at 1, this is always a multiple of 3
		// The first reduction step is one less
		step := hdr.Step - 1 + uint8(i)

		if step > config.ConsensusMaxStep {
			return fmt.Errorf("invalid step value")
		}

		// Committee the sortition determines for this round
		committee := a.Committee(hdr.Round, step)

		// subcommittee is a subset of the committee members that voted - the so-called quorum-committee
		subcommittee := committee.IntersectCluster(votes.BitSet)

		stepVoters := subcommittee.TotalOccurrences()

		log := consensus.WithFields(hdr.Round, step, "agreement_received",
			hdr.BlockHash, a.Keys.BLSPubKey, &committee, &subcommittee, &a.Provisioners)

		log.WithField("bitset", votes.BitSet).
			// number of committee members for current (round, step, seed) tuple
			WithField("comm", committee.Len()).
			// number of subcommittee members
			WithField("subcomm", subcommittee.Len()).
			// total votes.
			// NB if same provisioner is extracted multiple times per a step and
			// it does vote, stepVoters is higher than subcommittee.Len().
			WithField("t_votes", stepVoters).Info()

		// Verify quorum threshold reached
		if stepVoters < quorumTarget {
			return fmt.Errorf("vote set too small - %v/%v", stepVoters, quorumTarget)
		}

		// Verify aggregated signature is correct
		// a.k.a message (round, step, hash) is signed by all subcomittee members
		apk, err := AggregatePks(&a.Provisioners, subcommittee.Set)
		if err != nil {
			return fmt.Errorf("failed to reconstruct APK in the Agreement verification: %w", err)
		}

		if err := header.VerifySignatures(hdr.Round, step, hdr.BlockHash, apk, votes.Signature); err != nil {
			return fmt.Errorf("failed to verify BLS multisig: %w", err)
		}
	}

	return nil
}

func (a *handler) getVoterKeys(ev message.Agreement) ([][]byte, error) {
	hdr := ev.State()
	keys := make([][]byte, 0)

	for i, votes := range ev.VotesPerStep {
		step := hdr.Step - 1 + uint8(i)

		if step > config.ConsensusMaxStep {
			err := errors.New("getVoterKeys, step reached max limit")
			lg.WithError(err).Error("step overflow")

			return nil, err
		}

		committee := a.Committee(hdr.Round, step)
		subcommittee := committee.IntersectCluster(votes.BitSet)

		keys = append(keys, subcommittee.Unravel()...)
	}

	return keys, nil
}

func verifyWhole(a message.Agreement) error {
	hdr := a.State()

	r := new(bytes.Buffer)
	if err := header.MarshalSignableVote(r, hdr); err != nil {
		return err
	}

	return msg.VerifyBLSSignature(hdr.PubKeyBLS, a.Signature(), r.Bytes())
}

// AggregatePks reconstructs an aggregated BLS public key from a subcommittee.
func AggregatePks(p *user.Provisioners, subcommittee sortedset.Set) ([]byte, error) {
	if cfg.Get().Consensus.UseCompressedKeys {
		return aggregateCompressedPks(subcommittee)
	}

	return aggregateUncompressedPks(p, subcommittee)
}

// aggregateCompressedPks reconstructs compressed BLS public key.
func aggregateCompressedPks(subcommittee sortedset.Set) ([]byte, error) {
	var apk []byte
	var err error

	if len(subcommittee) == 0 {
		return nil, errors.New("Subcommittee is empty")
	}

	pks := make([][]byte, 0)

	for i, ipk := range subcommittee {
		pk := ipk.Bytes()

		if i == 0 {
			apk, err = bls.CreateApk(pk)
			if err != nil {
				return nil, err
			}

			continue
		}

		if len(pk) != 96 {
			panic("invalid pubkey size")
		}

		pks = append(pks, pk)
	}

	if len(pks) > 0 {
		// Instead of calling AggregatePk per each PubKey, we mitigate Cgo
		// overhead by aggregating a set of Pubkeys in a single call.
		// Benchmarking indicates 30% faster execution.
		apk, err = bls.AggregatePk(apk, pks...)
		if err != nil {
			return nil, err
		}
	}

	return apk, nil
}

// aggregateCompressedPks reconstructs uncompressed BLS public.
func aggregateUncompressedPks(p *user.Provisioners, subcommittee sortedset.Set) ([]byte, error) {
	var apk []byte
	var err error

	pks := make([][]byte, 0)

	for _, ipk := range subcommittee {
		rawPk := p.GetRawPublicKeyBLS(ipk.Bytes())
		if len(rawPk) != 0 {
			pks = append(pks, rawPk)
		}
	}

	if len(pks) == 0 {
		return nil, errors.New("empty committee")
	}

	apk, err = bls.AggregatePKsUnchecked(pks...)
	if err != nil {
		return nil, err
	}

	return apk, nil
}
