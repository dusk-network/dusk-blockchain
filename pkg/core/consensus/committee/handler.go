package committee

import (
	"math"
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
)

var PregenerationAmount uint8 = 8

type Handler struct {
	user.Keys
	Provisioners user.Provisioners
	Committees   []user.VotingCommittee
	lock         sync.RWMutex
}

func NewHandler(keys user.Keys, p user.Provisioners) *Handler {
	return &Handler{
		Keys:         keys,
		Committees:   make([]user.VotingCommittee, math.MaxUint8),
		Provisioners: p,
	}
}

// AmMember checks if we are part of the committee for a given round and step.
func (b *Handler) AmMember(round uint64, step uint8, maxSize int) bool {
	return b.IsMember(b.Keys.BLSPubKeyBytes, round, step, maxSize)
}

// IsMember checks if a provisioner with a given BLS public key is
// part of the committee for a given round and step.
func (b *Handler) IsMember(pubKeyBLS []byte, round uint64, step uint8, maxSize int) bool {
	return b.Committee(round, step, maxSize).IsMember(pubKeyBLS)
}

// Committee returns a VotingCommittee for a given round and step.
func (b *Handler) Committee(round uint64, step uint8, maxSize int) user.VotingCommittee {
	if b.membersAt(step-1) == 0 {
		b.generateCommittees(round, step, maxSize)
	}
	b.lock.RLock()
	committee := b.Committees[step-1]
	b.lock.RUnlock()
	return committee
}

func (b *Handler) generateCommittees(round uint64, step uint8, maxSize int) {
	size := b.CommitteeSize(maxSize)

	b.lock.Lock()
	defer b.lock.Unlock()
	committees := b.Provisioners.GenerateCommittees(round, PregenerationAmount, step, size)
	for i, committee := range committees {
		b.Committees[int(step)+i-1] = committee
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
