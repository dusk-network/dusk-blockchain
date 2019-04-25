package user

import (
	"bytes"
	"encoding/hex"
	"fmt"
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

	// Provisioners is a slice of Members, and makes up the current provisioner committee.
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

// GetMemberBLS returns a member of the provisioners from its BLS key
func (p *Provisioners) GetMemberBLS(pubKeyBLS []byte) *Member {
	for _, provisioner := range *p {
		if bytes.Equal(provisioner.PublicKeyBLS.Marshal(), pubKeyBLS) {
			return &provisioner
		}
	}
	return nil
}

// GetMemberEd returns a member of the provisioners from its Ed25519 key
func (p *Provisioners) GetMemberEd(pubKeyEd []byte) *Member {
	for _, provisioner := range *p {
		if bytes.Equal([]byte(provisioner.PublicKeyEd), pubKeyEd) {
			return &provisioner
		}
	}
	return nil
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
	for _, member := range *p {
		if m.EdEquals(member) {
			return nil
		}
	}

	*p = append(*p, m)

	// Sort the list
	p.sort()
	return nil
}

// RemoveMember will iterate over the committee and remove the specified Member.
func (p *Provisioners) RemoveMember(pubKeyBLS []byte) error {
	if len(pubKeyBLS) != 129 {
		return fmt.Errorf("public key is %v bytes long instead of 129", len(pubKeyBLS))
	}

	var m Member
	if err := m.PublicKeyBLS.Unmarshal(pubKeyBLS); err != nil {
		return err
	}

	for i, member := range *p {
		if m.BLSEquals(member) {
			list := *p
			list = append(list[:i], list[i+1:]...)
			*p = list
		}
	}

	// Sort the list
	p.sort()
	return nil
}

// GetStake will find a certain provisioner in the committee by BLS public key,
// and return their stake.
func (p Provisioners) GetStake(pubKeyBLS []byte) (uint64, error) {
	if len(pubKeyBLS) != 129 {
		return 0, fmt.Errorf("public key is %v bytes long instead of 128", len(pubKeyBLS))
	}

	var m Member
	pubKey := &bls.PublicKey{}
	if err := pubKey.Unmarshal(pubKeyBLS); err != nil {
		return 0, err
	}

	m.PublicKeyBLS = *pubKey

	for _, member := range p {
		if m.BLSEquals(member) {
			return member.Stake, nil
		}
	}

	return 0, nil
}

// Sort will sort the committee lexicographically
func (p *Provisioners) sort() {
	list := *p
	sort.SliceStable(list, func(i, j int) bool {
		return list[i].EdString() < list[j].EdString()
	})
	*p = list
}
