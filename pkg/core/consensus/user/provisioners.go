package user

import (
	"bytes"
	"encoding/hex"
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
	}

	// Provisioners is a slice of Members, and makes up the current provisioner committee. It implements sort.Interface
	Provisioners []Member
)

// EdEquals will check if the two passed Members are of the same value.
func (m Member) EdEquals(member Member) bool {
	return bytes.Equal([]byte(m.PublicKeyEd), []byte(member.PublicKeyEd))
}

// BLSEquals will check if the two passed Members are of the same value.
func (m Member) BLSEquals(member Member) bool {
	return bytes.Equal(m.PublicKeyBLS.Marshal(), member.PublicKeyBLS.Marshal())
}

// EdString returns the hexadecimal string representation of a Member
// Ed25519 public key.
func (m Member) EdString() string {
	return hex.EncodeToString([]byte(m.PublicKeyEd))
}

// BLSString returns the hexadecimal string representation of a Member
// BLS public key.
func (m Member) BLSString() string {
	return hex.EncodeToString(m.PublicKeyBLS.Marshal())
}

func (p *Provisioners) Len() int {
	return len(*p)
}

func (p *Provisioners) Swap(i, j int) {
	m := *p
	m[i], m[j] = m[j], m[i]
	p = &m
}

func (p *Provisioners) Less(i, j int) bool {
	m := *p
	mI, mJ := &big.Int{}, &big.Int{}
	mI.SetBytes(m[i].PublicKeyBLS.Marshal())
	mJ.SetBytes(m[j].PublicKeyBLS.Marshal())
	return mI.Cmp(mJ) < 0
}

// GetMemberBLS returns a member of the provisioners from its BLS key
func (p *Provisioners) GetMemberBLS(pubKeyBLS []byte) *Member {
	i, found := p.IndexOf(pubKeyBLS)
	if !found {
		return nil
	}

	return &(*p)[i]
}

// AddMember will add a Member to the Provisioners by using the bytes of an Ed25519  public key.
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

	// Check for duplicates
	i, found := p.IndexOf(pubKeyBLS)
	if found {
		(*p)[i].Stake = stake
		return nil
	}
	list := *p

	// inserting member at index i
	*p = append(list[:i], append([]Member{m}, list[:i+1]...)...)
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

	list := *p
	list = append(list[:i], list[i+1:]...)
	*p = list
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

	iRepr := &big.Int{}
	i := sort.Search(len(*p), func(i int) bool {
		bRepr := (*p)[i].PublicKeyBLS.Marshal()
		iRepr.SetBytes(bRepr)

		cmp := iRepr.Cmp(iPk)
		if cmp == 0 {
			found = true
		}
		return cmp >= 0
	})

	return i, found
}
