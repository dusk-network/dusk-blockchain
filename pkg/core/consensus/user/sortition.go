package user

import (
	"encoding/binary"
	"math"
	"math/big"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/wallet"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/sortedset"
	"github.com/dusk-network/dusk-crypto/hash"
	log "github.com/sirupsen/logrus"
)

// VotingCommittee represents a set of provisioners with voting rights at a certain
// point in the consensus. The set is sorted by the int value of the public key in
// increasing order (higher last)
type VotingCommittee struct {
	sortedset.Cluster
}

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

// Equal checks if two VotingCommittees are the same.
func (v VotingCommittee) Equal(other *VotingCommittee) bool {
	return v.Cluster.Equal(other.Cluster)
}

// IsMember checks if `pubKeyBLS` is within the VotingCommittee.
func (v VotingCommittee) IsMember(pubKeyBLS []byte) bool {
	_, found := v.IndexOf(pubKeyBLS)
	return found
}

// createSortitionMessage will return the hash of the passed sortition information.
func createSortitionHash(round uint64, step uint8, i int) ([]byte, error) {
	msg := make([]byte, 12)
	binary.LittleEndian.PutUint64(msg[:8], round)
	binary.LittleEndian.PutUint32(msg[8:12], uint32(i))
	msg = append(msg, step)
	return hash.Sha3256(msg)
}

// generate a score from the given hash and total stake weight
func generateSortitionScore(hash []byte, W *big.Int) uint64 {
	hashNum := new(big.Int).SetBytes(hash)
	return new(big.Int).Mod(hashNum, W).Uint64()
}

// CreateVotingCommittee will run the deterministic sortition function, which determines
// who will be in the committee for a given step and round.
// FIXME: running this with weird setup causes infinite looping (to reproduce, hardcode `3` on MockProvisioners when calling agreement.NewHelper in the agreement tests)
func (p Provisioners) CreateVotingCommittee(round uint64, step uint8, size int) VotingCommittee {
	votingCommittee := newCommittee()
	W := new(big.Int).SetUint64(p.TotalWeight())
	// Deep copy the Members map, to avoid mutating the original set.
	members := copyMembers(p.Members)
	p.Members = members

	// Remove stakes which have not yet become active, or have expired
	for _, m := range p.Members {
		i := 0
		for {
			if i == len(m.Stakes) {
				break
			}

			if m.Stakes[i].StartHeight > round || m.Stakes[i].EndHeight < round {
				subtractFromTotalWeight(W, m.Stakes[i].Amount)
				m.RemoveStake(i)
				continue
			}

			i++
		}
	}

	for i := 0; votingCommittee.Size() < size; i++ {
		if W.Uint64() == 0 {
			// We ran out of staked DUSK, so we return the result prematurely
			break
		}

		hash, err := createSortitionHash(round, step, i)
		if err != nil {
			log.Panic(err)
		}

		score := generateSortitionScore(hash, W)
		blsPk := p.extractCommitteeMember(score)
		votingCommittee.Insert(blsPk)

		// Subtract up to one DUSK from the extracted committee member.
		m := p.GetMember(blsPk)
		subtracted := m.SubtractFromStake(1 * wallet.DUSK)

		// Also subtract the subtracted amount from the total weight, to ensure
		// consistency.
		subtractFromTotalWeight(W, subtracted)
	}

	return *votingCommittee
}

// extractCommitteeMember walks through the committee set, while deducting
// each node's stake from the passed score until we reach zero. The public key
// of the node that the function ends on will be returned as a hexadecimal string.
func (p Provisioners) extractCommitteeMember(score uint64) []byte {
	var m *Member
	var e error
	for i := 0; ; i++ {
		if m, e = p.MemberAt(i); e != nil {
			// handling the eventuality of an out of bound error
			m, e = p.MemberAt(0)
			if e != nil {
				log.Panic(e)
			}
		}

		stake, err := p.GetStake(m.PublicKeyBLS)
		if err != nil {
			// If we get an error from GetStake, it means we either got a public key of a
			// provisioner who is no longer in the set, or we got a malformed public key.
			// We can't repair our committee on the fly, so we have to panic.
			log.Panic(err)
		}

		if stake >= score {
			return m.PublicKeyBLS
		}

		score -= stake
	}
}

// GenerateCommittees pre-generates an `amount` of VotingCommittee of a specified `size` from a given `step`
func (p Provisioners) GenerateCommittees(round uint64, amount, step uint8, size int) []VotingCommittee {
	if step >= math.MaxUint8-amount {
		amount = math.MaxUint8 - step
	}

	committees := make([]VotingCommittee, amount)
	for i := 0; i < int(amount); i++ {
		votingCommittee := p.CreateVotingCommittee(round, step+uint8(i), size)
		committees[i] = votingCommittee
	}

	return committees
}

func subtractFromTotalWeight(W *big.Int, amount uint64) {
	if W.Uint64() > amount {
		W.Sub(W, big.NewInt(int64(amount)))
		return
	}

	W.Set(big.NewInt(0))
}

// Deep copy a Members map. Since slices are treated as 'reference types' by Go, we
// need to iterate over, and individually copy each Stake to the new Member struct,
// to avoid mutating the original set.
func copyMembers(members map[string]*Member) map[string]*Member {
	m := make(map[string]*Member)
	for k, v := range members {
		member := &Member{
			PublicKeyEd:  v.PublicKeyEd,
			PublicKeyBLS: v.PublicKeyBLS,
		}

		member.Stakes = append(member.Stakes, v.Stakes...)
		m[k] = member
	}

	return m
}
