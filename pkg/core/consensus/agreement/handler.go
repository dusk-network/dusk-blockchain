package agreement

import (
	"bytes"
	"errors"
	"fmt"
	"math"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/committee"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/msg"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/sortedset"
	"github.com/dusk-network/dusk-crypto/bls"
	"github.com/dusk-network/dusk-wallet/key"
)

const MaxCommitteeSize = 64

// Handler interface is handy for tests
type Handler interface {
	AmMember(uint64, uint8) bool
	IsMember([]byte, uint64, uint8) bool
	Committee(uint64, uint8) user.VotingCommittee
	Quorum() int
	VotesFor([]byte, uint64, uint8) int
	Verify(Agreement) error
}

type handler struct {
	*committee.Handler
}

// newHandler returns an initialized handler.
func newHandler(keys key.ConsensusKeys, p user.Provisioners) *handler {
	return &handler{
		Handler: committee.NewHandler(keys, p),
	}
}

// AmMember checks if we are part of the committee.
func (a *handler) AmMember(round uint64, step uint8) bool {
	return a.Handler.AmMember(round, step, MaxCommitteeSize)
}

func (a *handler) IsMember(pubKeyBLS []byte, round uint64, step uint8) bool {
	return a.Handler.IsMember(pubKeyBLS, round, step, MaxCommitteeSize)
}

func (a *handler) Committee(round uint64, step uint8) user.VotingCommittee {
	return a.Handler.Committee(round, step, MaxCommitteeSize)
}

func (a *handler) VotesFor(pubKeyBLS []byte, round uint64, step uint8) int {
	return a.Handler.VotesFor(pubKeyBLS, round, step, MaxCommitteeSize)
}

func (a *handler) Quorum() int {
	return int(math.Ceil(float64(a.CommitteeSize(MaxCommitteeSize)) * 0.75))
}

// Verify checks the signature of the set.
func (a *handler) Verify(ev Agreement) error {
	if err := verifyWhole(ev); err != nil {
		return err
	}

	allVoters := 0
	for i, votes := range ev.VotesPerStep {
		step := ev.Step - 2 + uint8(i)
		committee := a.Committee(ev.Round, step)
		subcommittee := committee.Intersect(votes.BitSet)

		allVoters += len(subcommittee)
		apk, err := ReconstructApk(subcommittee)
		if err != nil {
			return err
		}

		if err := header.VerifySignatures(ev.Round, step, ev.BlockHash, apk, votes.Signature); err != nil {
			return err
		}
	}

	if allVoters < a.Quorum() {
		return fmt.Errorf("vote set too small - %v/%v", allVoters, a.Quorum())
	}
	return nil
}

func verifyWhole(a Agreement) error {
	r := new(bytes.Buffer)
	if err := header.MarshalSignableVote(r, a.Header); err != nil {
		return err
	}

	return msg.VerifyBLSSignature(a.Header.PubKeyBLS, r.Bytes(), a.SignedVotes())
}

// ReconstructApk reconstructs an aggregated BLS public key from a subcommittee.
func ReconstructApk(subcommittee sortedset.Set) (*bls.Apk, error) {
	var apk *bls.Apk
	if len(subcommittee) == 0 {
		return nil, errors.New("Subcommittee is empty")
	}

	// We need to avoid adding duplicates to the APK, as it will cause verification to fail.
	for i, ipk := range subcommittee {
		pk, err := bls.UnmarshalPk(ipk.Bytes())
		if err != nil {
			return nil, err
		}
		if i == 0 {
			apk = bls.NewApk(pk)
			continue
		}
		if err := apk.Aggregate(pk); err != nil {
			return nil, err
		}
	}

	return apk, nil
}
