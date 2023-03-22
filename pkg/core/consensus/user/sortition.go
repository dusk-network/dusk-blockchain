// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package user

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"math/big"

	"github.com/dusk-network/dusk-blockchain/pkg/util"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/sortedset"
	"github.com/dusk-network/dusk-crypto/hash"
	log "github.com/sirupsen/logrus"
)

// DUSK is one whole unit of DUSK. This is duplicated from wallet since
// otherwise we get into an import cycle including the transactions and users
// packages.
const DUSK = uint64(100000000)

// VotingCommittee represents a set of provisioners with voting rights at a certain
// point in the consensus. The set is sorted by the int value of the public key in
// increasing order (higher last).
type VotingCommittee struct {
	sortedset.Cluster
}

// newCommittee creates a new VotingCommittee set.
func newCommittee() *VotingCommittee {
	return &VotingCommittee{
		Cluster: sortedset.NewCluster(),
	}
}

// Size returns how many members there are in a VotingCommittee.
func (v VotingCommittee) Size() int {
	return v.TotalOccurrences()
}

// MemberKeys returns the BLS public keys of all the members in a VotingCommittee.
func (v VotingCommittee) MemberKeys() [][]byte {
	return v.Unravel()
}

// Equal checks if two VotingCommittees are equal (i.e. they contain the same set of provisioners).
func (v VotingCommittee) Equal(other *VotingCommittee) bool {
	return v.Cluster.Equal(other.Cluster)
}

// IsMember checks if `pubKeyBLS` is within the VotingCommittee.
func (v VotingCommittee) IsMember(pubKeyBLS []byte) bool {
	_, found := v.IndexOf(pubKeyBLS)
	return found
}

// Format implements fmt.Formatter interface.
func (v VotingCommittee) Format(f fmt.State, c rune) {
	r := fmt.Sprintf("cluster: %v", v.Cluster)
	_, _ = f.Write([]byte(r))
}

// MarshalJSON allows to print VotingCommittee list in JSONFormatter.
func (v VotingCommittee) MarshalJSON() ([]byte, error) {
	data := make([]string, 0)

	for _, bi := range v.Set {
		r := fmt.Sprintf("Key: %s, Count: %d", util.StringifyBytes(bi.Bytes()), v.Cluster.OccurrencesOf(bi.Bytes()))
		data = append(data, r)
	}

	return json.Marshal(data)
}

// createSortitionHash takes the Seed value 'seed', the round number 'round', the step number 'step',
// the index 'i' of the committee member to extract,
// and returns the hash (SHA3-256) of their concatenation (i.e., H(round||i||step||seed)).
func createSortitionHash(seed []byte, round uint64, step uint8, i int) ([]byte, error) {
	msg := make([]byte, 12)

	binary.LittleEndian.PutUint64(msg[:8], round)
	binary.LittleEndian.PutUint32(msg[8:12], uint32(i))

	msg = append(msg, step)
	msg = append(msg, seed...)

	return hash.Sha3256(msg)
}

// generateSortitionScore generates a score value from the sortition hash 'hash' and the total stake weight 'W'.
// It returns score=(hashNum % W), where 'hashNum' is the integer interpretation of 'hash'.
func generateSortitionScore(hash []byte, W *big.Int) uint64 {
	hashNum := new(big.Int).SetBytes(hash)
	return new(big.Int).Mod(hashNum, W).Uint64()
}

// CreateVotingCommittee executes the Deterministic Sortition algorithm
// to determine the committee members for a given step and round.
// TODO: running this with weird setup causes infinite looping (to reproduce, hardcode `3` on MockProvisioners when calling agreement.NewHelper in the agreement tests).
func (p Provisioners) CreateVotingCommittee(seed []byte, round uint64, step uint8, size int) VotingCommittee {
	votingCommittee := newCommittee()
	W := new(big.Int).SetUint64(p.TotalWeight())

	// Deep copy the Members map, to avoid mutating the original set.
	members := copyMembers(p.Members)
	p.Members = members

	// Remove stakes which have not yet "mature"
	for _, m := range p.Members {
		i := 0

		for {
			if i == len(m.Stakes) {
				break
			}

			isStakeMature := m.Stakes[i].Eligibility <= round
			if !isStakeMature {
				subtractFromTotalWeight(W, m.Stakes[i].Value)
				m.RemoveStake(i)
				continue
			}

			i++
		}
	}

	// Build votingCommittee, adding one extracted provisioner at a time
	// From each member, we deduct up to 1 DUSK from their stake
	for i := 0; votingCommittee.Size() < size; i++ {
		// If we run out of staked DUSK, we can't add new members to the committee
		// If this happens, we leave the votingCommittee partially complete
		if W.Uint64() == 0 {
			break
		}

		// Create Sortition Hash
		hashSort, err := createSortitionHash(seed, round, step, i)
		if err != nil {
			log.Panic(err)
		}

		// Generate Score
		score := generateSortitionScore(hashSort, W)

		// Extract new committee member
		blsPk := p.extractCommitteeMember(score)
		votingCommittee.Insert(blsPk)

		// Deduct up to 1 DUSK from the extracted member's stake.
		m := p.GetMember(blsPk)
		subtracted := m.SubtractFromStake(1 * DUSK)

		// Subtract the deducted amount from the total weight, to ensure consistency.
		subtractFromTotalWeight(W, subtracted)
	}

	return *votingCommittee
}

// extractCommitteeMember walks through the provisioners set, while deducting each stake
// from the sortition 'score', until this is lower than the current stake.
// When this occurs, it returns the BLS key of the provisioner on which it stops (i.e. the extracted member).
func (p Provisioners) extractCommitteeMember(score uint64) []byte {
	var m *Member
	var e error

	for i := 0; ; i++ {
		// If a provisioner is missing, we use the provisioner at position 0
		if m, e = p.MemberAt(i); e != nil {
			m, e = p.MemberAt(0)

			// If provisioner 0 is also missing, panic
			if e != nil {
				// FIXME: shall this panic?
				log.Panic(e)
			}

			i = 0
		}

		stake, err := p.GetStake(m.PublicKeyBLS)
		if err != nil {
			// If we get an error from GetStake, it means we either got a public key of a
			// provisioner who is no longer in the set, or we got a malformed public key.
			// We can't repair our committee on the fly, so we have to panic.
			log.Panic(fmt.Errorf("pk: %s err: %v", util.StringifyBytes(m.PublicKeyBLS), err))
		}

		// If the current stake is higher than the score, return the current provisioner's BLS key
		if stake >= score {
			return m.PublicKeyBLS
		}

		score -= stake
	}
}

// GenerateCommittees pre-generates an `amount` of voting committees of a specified 'size', starting from step 'step'.
func (p Provisioners) GenerateCommittees(seed []byte, round uint64, amount, step uint8, size int) []VotingCommittee {
	if step >= math.MaxUint8-amount {
		amount = math.MaxUint8 - step
	}

	committees := make([]VotingCommittee, amount)

	// Create 'amount' voting committees of size 'size' for steps between 'step' and 'step'+('amount'-1)
	for i := 0; i < int(amount); i++ {
		votingCommittee := p.CreateVotingCommittee(seed, round, step+uint8(i), size)
		committees[i] = votingCommittee
	}

	return committees
}

// Format implements fmt.Formatter interface.
// Prints all members and its stakes.
func (p Provisioners) Format(f fmt.State, c rune) {
	for _, m := range p.Members {
		r := fmt.Sprintf("BLS key: %s, Stakes: %q", util.StringifyBytes(m.PublicKeyBLS), m.Stakes)
		_, _ = f.Write([]byte(r))
	}
}

// MarshalJSON allows to print Provisioners list in JSONFormatter.
func (p Provisioners) MarshalJSON() ([]byte, error) {
	data := make([]string, 0)

	for _, m := range p.Members {
		r := fmt.Sprintf("BLS key: %s, Stakes: %q", util.StringifyBytes(m.PublicKeyBLS), m.Stakes)
		data = append(data, r)
	}

	return json.Marshal(data)
}

// subtractFromTotalWeight subtracts 'amount' from the total weight 'W'.
// If 'amount' is bigger than 'W', it sets 'W' to 0.
func subtractFromTotalWeight(W *big.Int, amount uint64) {
	if W.Uint64() > amount {
		W.Sub(W, big.NewInt(int64(amount)))
		return
	}

	// If 'amount' is bigger than 'W', set 'W' to 0
	W.Set(big.NewInt(0))
}

// Deep copy a Members map. Since slices are treated as 'reference types' by Go, we
// need to iterate over, and individually copy each Stake to the new Member struct,
// to avoid mutating the original set.
func copyMembers(members map[string]*Member) map[string]*Member {
	m := make(map[string]*Member)

	for k, v := range members {
		m[k] = v.Copy()
	}

	return m
}
