package user

import (
	"fmt"
	"math/big"
	"sort"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"golang.org/x/crypto/ed25519"
)

// Member contains the bytes of a provisioner's Ed25519 public key,
// the bytes of his BLS public key, and how much he has staked.
type (
	Member struct {
		PublicKeyEd  ed25519.PublicKey
		PublicKeyBLS bls.PublicKey
		Stake        uint64
		blsId        *big.Int
	}

	// Provisioners is a slice of Members, and makes up the current provisioner committee. It implements sort.Interface
	Provisioners []Member
)

func (p *Provisioners) Len() int {
	return len(*p)
}

func (p *Provisioners) Swap(i, j int) {
	(*p)[i], (*p)[j] = (*p)[j], (*p)[i]
}

func (p *Provisioners) Less(i, j int) bool {
	mI, mJ := &big.Int{}, &big.Int{}
	mI.SetBytes((*p)[i].PublicKeyBLS.Marshal())
	mJ.SetBytes((*p)[j].PublicKeyBLS.Marshal())
	return mI.Cmp(mJ) < 0
}

// GetMemberBLS returns a member of the provisioners from its BLS key
func (p *Provisioners) GetMember(pubKeyBLS []byte) *Member {
	i, found := p.IndexOf(pubKeyBLS)
	if !found {
		return nil
	}

	return &(*p)[i]
}

// AddMember will add a Member to the Provisioners by using the bytes of a BLS public key. Returns the index at which the key has been inserted. If the member alredy exists, AddMember overrides its stake with the new one
func (p *Provisioners) AddMember(pubKeyEd, pubKeyBLS []byte, stake uint64) error {
	if len(pubKeyEd) != 32 {
		return fmt.Errorf("public key is %v bytes long instead of 32", len(pubKeyEd))
	}

	if len(pubKeyBLS) != 129 {
		return fmt.Errorf("public key is %v bytes long instead of 128", len(pubKeyBLS))
	}

	var m Member
	m.PublicKeyEd = ed25519.PublicKey(pubKeyEd)

	pubKey := &bls.PublicKey{}
	if err := pubKey.Unmarshal(pubKeyBLS); err != nil {
		return err
	}

	m.PublicKeyBLS = *pubKey
	m.Stake = stake
	iPk := &big.Int{}
	iPk.SetBytes(pubKeyBLS)
	m.blsId = iPk

	// Check for duplicates
	i, found := p.IndexOf(pubKeyBLS)
	if found {
		(*p)[i].Stake = stake
		return nil
	}

	// inserting member at index i
	*p = append((*p)[:i], append([]Member{m}, (*p)[i:]...)...)
	return nil
}

// RemoveMember will iterate over the committee and remove the specified Member.
func (p *Provisioners) RemoveMember(pubKeyBLS []byte) error {
	if len(pubKeyBLS) != 129 {
		return fmt.Errorf("public key is %v bytes long instead of 129", len(pubKeyBLS))
	}

	i, found := p.IndexOf(pubKeyBLS)
	if !found {
		return fmt.Errorf("public %v not found among provisioner set", pubKeyBLS)
	}

	*p = append((*p)[:i], (*p)[i+1:]...)
	return nil
}

// GetStake will find a certain provisioner in the committee by BLS public key,
// and return their stake.
func (p *Provisioners) GetStake(pubKeyBLS []byte) (uint64, error) {
	if len(pubKeyBLS) != 129 {
		return 0, fmt.Errorf("public key is %v bytes long instead of 128", len(pubKeyBLS))
	}

	i, found := p.IndexOf(pubKeyBLS)
	if !found {
		return 0, fmt.Errorf("public %v not found among provisioner set", pubKeyBLS)
	}

	return (*p)[i].Stake, nil
}

// IndexOf performs a binary search of the Provisioners and returns the index where the Provisioner identified by this blsPk would be inserted at and a bool indicating whether the Provisioner is already in the array or not
func (p *Provisioners) IndexOf(blsPk []byte) (int, bool) {
	found := false
	iPk := &big.Int{}
	iPk.SetBytes(blsPk)

	i := sort.Search(len(*p), func(i int) bool {
		iRepr := (*p)[i].blsId
		cmp := iRepr.Cmp(iPk)
		if cmp == 0 {
			found = true
		}
		return cmp >= 0
	})

	return i, found
}
