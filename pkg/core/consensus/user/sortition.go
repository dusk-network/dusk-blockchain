package user

import (
	"encoding/binary"
	"math/big"
	"sort"
	"strconv"
	"strings"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/hash"
)

// VotingCommittee represents a set of provisioners with voting rights at a certain
// point in the consensus. The set is sorted by the int value of the public key in increasing order (higher last)
type VotingCommittee []*big.Int

func (v VotingCommittee) Len() int           { return len(v) }
func (v VotingCommittee) Swap(i, j int)      { v[i], v[j] = v[j], v[i] }
func (v VotingCommittee) Less(i, j int) bool { return v[i].Cmp(v[j]) < 0 }

func newCommittee() VotingCommittee {
	return make([]*big.Int, 0)
}

func (v VotingCommittee) Equal(other VotingCommittee) bool {
	sort.Sort(other)
	for i := 0; i < len(v); i++ {
		if v[i].Cmp(other[i]) != 0 {
			return false
		}
	}

	return true
}

// Intersect the bit representation of a VotingCommittee subset with the whole VotingCommittee set
func (v VotingCommittee) Intersect(committeeSet uint64) VotingCommittee {
	elems := newCommittee()
	for i, elem := range v {
		// looping on all bits to see which one is set to 1
		if ((committeeSet >> uint(i)) & 1) != 0 {
			elems = append(elems, elem)
		}
	}
	return elems
}

// Bits creates a bit representation of the subset of a VotingCommittee. It is used to make communications more efficient and avoid sending the list of signee of a committee
func (v VotingCommittee) Bits(pks VotingCommittee) uint64 {
	ret := uint64(0)
	if len(pks) == 0 {
		return ret
	}

	sort.Sort(pks)
	var head *big.Int
	head, pks = pks[0], pks[1:]
	for i, elem := range v {
		if elem.Cmp(head) == 0 {
			ret |= (1 << uint(i)) // flip the i-th bit to 1
			if len(pks) == 0 {
				break
			}
			head, pks = pks[0], pks[1:]
		}
	}
	return ret
}

func (v VotingCommittee) IndexOf(blsPk []byte) (int, bool) {
	cmp := 0
	found := false
	iPk := &big.Int{}
	iPk.SetBytes(blsPk)

	i := sort.Search(len(v), func(i int) bool {
		cmp = v[i].Cmp(iPk)
		if cmp == 0 {
			found = true
		}
		return cmp >= 0
	})
	return i, found
}

func (v *VotingCommittee) Remove(pubKeyBLS []byte) {
	i, found := v.IndexOf(pubKeyBLS)
	if found {
		*v = append((*v)[:i], (*v)[i+1:]...)
	}
}

func (v VotingCommittee) String() string {
	var str strings.Builder
	for i, bi := range v {
		str.WriteString("idx: ")
		str.WriteString(strconv.Itoa(i))
		str.WriteString(" nr: ")
		str.WriteString(shortStr(bi))
		str.WriteString("\n")
	}
	return str.String()
}

// IsMember checks if `pubKeyBLS` is within the VotingCommittee.
// Deprecated: use VotingCommittee.Bits and VotingCommittee.Intersect for transmission of VotingCommittee. Use VotingCommittee.IndexOf for member checking
func (v VotingCommittee) IsMember(pubKeyBLS []byte) bool {
	_, found := v.IndexOf(pubKeyBLS)
	return found
}

// VotingCommitteeSize returns how big the voting committee should be.
func (p Provisioners) VotingCommitteeSize() int {
	size := len(p)
	if size > 50 {
		size = 50
	}

	return size
}

// createSortitionMessage will return the hash of the passed sortition information.
func createSortitionHash(round uint64, step uint8, i int) ([]byte, error) {
	msg := make([]byte, 12)
	binary.LittleEndian.PutUint64(msg[:8], round)
	binary.LittleEndian.PutUint32(msg[8:12], uint32(i))
	msg = append(msg, byte(step))
	return hash.Sha3256(msg)
}

// generate a score from the given hash and total stake weight
func generateSortitionScore(hash []byte, W *big.Int) uint64 {
	hashNum := new(big.Int).SetBytes(hash)
	return new(big.Int).Mod(hashNum, W).Uint64()
}

// CreateVotingCommittee will run the deterministic sortition function, which determines
// who will be in the committee for a given step and round.
func (p Provisioners) CreateVotingCommittee(round, totalWeight uint64,
	step uint8) VotingCommittee {

	votingCommittee := newCommittee()
	W := new(big.Int).SetUint64(totalWeight)
	size := p.VotingCommitteeSize()

	for i := 0; len(votingCommittee) < size; i++ {
		hash, err := createSortitionHash(round, step, i)
		if err != nil {
			panic(err)
		}

		score := generateSortitionScore(hash, W)
		blsPk := p.extractCommitteeMember(score)
		votingCommittee.Insert(blsPk)
	}

	return votingCommittee
}

// extractCommitteeMember walks through the committee set, while deducting
// each node's stake from the passed score until we reach zero. The public key
// of the node that the function ends on will be returned as a hexadecimal string.
func (p Provisioners) extractCommitteeMember(score uint64) *big.Int {
	for i := 0; ; i++ {
		// make sure we wrap around the provisioners array
		if i == len(p) {
			i = 0
		}

		if p[i].Stake >= score {
			return p[i].blsId
		}

		score -= p[i].Stake
	}
}

// Insert a big.Int representation of a BLS key at a proper index (respectful of the VotingCommittee order). If the element is already in the VotingCommittee does nothing and returns false
func (v *VotingCommittee) Insert(blsPk *big.Int) bool {
	i, found := v.IndexOf(blsPk.Bytes())
	if !found {
		*v = append((*v)[:i], append([]*big.Int{blsPk}, (*v)[i:]...)...)
		return true
	}
	return false
}

func shortStr(i *big.Int) string {
	var str strings.Builder
	iStr := i.String()
	str.WriteString(iStr[:3])
	str.WriteString("...")
	str.WriteString(iStr[len(iStr)-3:])
	return str.String()
}
