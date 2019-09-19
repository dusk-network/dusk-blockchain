package committee

import (
	"math"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
)

var PregenerationAmount uint8 = 8

type Handler struct {
	user.Keys
	Provisioners user.Provisioners
	Committees   []user.VotingCommittee
}

func NewHandler(keys user.Keys) *Handler {
	return &Handler{
		Keys:       keys,
		Committees: make([]user.VotingCommittee, math.MaxUint8),
	}
}

// AmMember checks if we are part of the committee.
func (b *Handler) AmMember(round uint64, step uint8, maxSize int) bool {
	return b.IsMember(b.Keys.BLSPubKeyBytes, round, step, maxSize)
}

func (b *Handler) IsMember(pubKeyBLS []byte, round uint64, step uint8, maxSize int) bool {
	return b.Committee(round, step, maxSize).IsMember(pubKeyBLS)
}

func (b *Handler) Committee(round uint64, step uint8, maxSize int) user.VotingCommittee {
	if b.Committees[step-1].Set.Len() == 0 {
		committees := b.Provisioners.GenerateCommittees(round, PregenerationAmount, step, b.CommitteeSize(maxSize))
		for i, committee := range committees {
			b.Committees[int(step)+i-1] = committee
		}
	}

	return b.Committees[step-1]
}

func (b *Handler) CommitteeSize(maxSize int) int {
	size := len(b.Provisioners.Members)
	if size > maxSize {
		return maxSize
	}

	return size
}
