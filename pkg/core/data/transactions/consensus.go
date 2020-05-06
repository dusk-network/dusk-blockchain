package transactions

import (
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
)

// UMember deep copies from the rusk.Provisioner
func UMember(r *rusk.Provisioner, t *user.Member) {
	t.PublicKeyBLS = make([]byte, len(r.BlsKey))
	copy(t.PublicKeyBLS, r.BlsKey)
	t.Stakes = make([]user.Stake, len(r.Stakes))
	for i := range r.Stakes {
		t.Stakes[i] = user.Stake{
			Amount:      r.Stakes[i].Amount,
			StartHeight: r.Stakes[i].StartHeight,
			EndHeight:   r.Stakes[i].EndHeight,
		}
	}
}

// UBidList turns the rusk BidList into transactions equivalent
func UBidList(r *rusk.BidList, t *user.BidList) {
	/*
		t.BidList = make([][]byte, len(r.BidList))
		for i := range r.BidList {
			t.BidList[i] = make([]byte, len(r.BidList[i]))
			copy(t.BidList[i], r.BidList[i])
		}
	*/
}
