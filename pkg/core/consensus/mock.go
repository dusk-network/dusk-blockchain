package consensus

import (
	"bytes"
	"encoding/binary"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	crypto "github.com/dusk-network/dusk-crypto/hash"
)

func MockRoundUpdateBuffer(round uint64, p *user.Provisioners, bidList user.BidList) *bytes.Buffer {
	init := make([]byte, 8)
	binary.LittleEndian.PutUint64(init, round)
	buf := bytes.NewBuffer(init)

	if p == nil {
		p, _ = MockProvisioners(1)
	}
	user.MarshalProvisioners(buf, p)

	if bidList == nil {
		bidList = MockBidList()
	}
	user.MarshalBidList(buf, bidList)

	return buf
}

func MockProvisioners(amount int) (*user.Provisioners, []user.Keys) {
	p := user.NewProvisioners()

	k := make([]user.Keys, amount)
	for i := 0; i < amount; i++ {
		keys, _ := user.NewRandKeys()
		member := &user.Member{}
		member.PublicKeyEd = keys.EdPubKeyBytes
		member.PublicKeyBLS = keys.BLSPubKeyBytes
		member.Stakes = make([]user.Stake, 1)
		member.Stakes[0].Amount = 500
		member.Stakes[0].EndHeight = 10000

		p.Members[string(keys.BLSPubKeyBytes)] = member
		p.Set.Insert(keys.BLSPubKeyBytes)
		k[i] = keys
	}

	return p, k
}

func MockBidList() user.BidList {
	bidList := make([]user.Bid, 1)
	xSlice, _ := crypto.RandEntropy(32)
	mSlice, _ := crypto.RandEntropy(32)
	var x [32]byte
	var m [32]byte
	copy(x[:], xSlice)
	copy(m[:], mSlice)
	bidList[0] = user.Bid{x, m, 10}
	return bidList
}
