package transactions

import "github.com/dusk-network/dusk-protobuf/autogen/go/rusk"

// StakeData represents the Provisioner's stake
type StakeData struct {
	Amount      uint64 `json:"amount"`
	StartHeight uint64 `json:"start_height"`
	EndHeight   uint64 `json:"end_height"`
}

// BidList is used to create the Proof of Blind Bid
type BidList struct {
	BidList [][]byte `json:"bid_list"`
}

// ProvisionerData is a representation of a Committee participant
// identified by its BLSKey and Stake
type ProvisionerData struct {
	BlsKey []byte       `json:"bls_key"`
	Stakes []*StakeData `json:"stakes"`
}

// UProvisioner deep copies from the rusk.Provisioner
func UProvisioner(r *rusk.Provisioner, t *ProvisionerData) {
	t.BlsKey = make([]byte, len(r.BlsKey))
	copy(t.BlsKey, r.BlsKey)
	t.Stakes = make([]*StakeData, len(r.Stakes))
	for i := range r.Stakes {
		UStakeData(r.Stakes[i], t.Stakes[i])
	}
}

// UBidList turns the rusk BidList into transactions equivalent
func UBidList(r *rusk.BidList, t *BidList) {
	t.BidList = make([][]byte, len(r.BidList))
	for i := range r.BidList {
		t.BidList[i] = make([]byte, len(r.BidList[i]))
		copy(t.BidList[i], r.BidList[i])
	}
}

// UStakeData turns rusk.Stake into its Transaction equivalent
func UStakeData(r *rusk.Stake, t *StakeData) {
	t.Amount = r.Amount
	t.StartHeight = r.StartHeight
	t.EndHeight = r.EndHeight
}
