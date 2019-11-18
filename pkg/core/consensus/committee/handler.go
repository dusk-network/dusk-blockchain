package committee

import (
	"math"
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-wallet/key"
)

var PregenerationAmount uint8 = 8

type Handler struct {
	key.ConsensusKeys
	Provisioners user.Provisioners
	Committees   []user.VotingCommittee
	lock         sync.RWMutex
}

func NewHandler(keys key.ConsensusKeys, p user.Provisioners) *Handler {
	return &Handler{
		ConsensusKeys: keys,
		Committees:    make([]user.VotingCommittee, math.MaxUint8),
		Provisioners:  p,
	}
}

// AmMember checks if we are part of the committee for a given round and step.
func (b *Handler) AmMember(round uint64, step uint8, maxSize int) bool {
	return b.IsMember(b.ConsensusKeys.BLSPubKeyBytes, round, step, maxSize)
}

// IsMember checks if a provisioner with a given BLS public key is
// part of the committee for a given round and step.
func (b *Handler) IsMember(pubKeyBLS []byte, round uint64, step uint8, maxSize int) bool {
	return b.Committee(round, step, maxSize).IsMember(pubKeyBLS)
}

// VotesFor returns the amount of votes for a public key for a given round and step.
func (b *Handler) VotesFor(pubKeyBLS []byte, round uint64, step uint8, maxSize int) int {
	return b.Committee(round, step, maxSize).OccurrencesOf(pubKeyBLS)
}

// Committee returns a VotingCommittee for a given round and step.
func (b *Handler) Committee(round uint64, step uint8, maxSize int) user.VotingCommittee {
	if b.membersAt(step) == 0 {
		b.generateCommittees(round, step, maxSize)
	}
	b.lock.RLock()
	committee := b.Committees[step]
	b.lock.RUnlock()
	return committee
}

func (b *Handler) generateCommittees(round uint64, step uint8, maxSize int) {
	size := b.CommitteeSize(maxSize)

	b.lock.Lock()
	defer b.lock.Unlock()
	committees := b.Provisioners.GenerateCommittees(round, PregenerationAmount, step, size)
	for i, committee := range committees {
		b.Committees[int(step)+i] = committee
	}
}

// CommitteeSize returns the size of a VotingCommittee, depending on
// how many provisioners are in the set.
func (b *Handler) CommitteeSize(maxSize int) int {
	b.lock.RLock()
	size := len(b.Provisioners.Members)
	b.lock.RUnlock()
	if size > maxSize {
		return maxSize
	}

	return size
}

func (b *Handler) membersAt(idx uint8) int {
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.Committees[idx].Set.Len()
}
