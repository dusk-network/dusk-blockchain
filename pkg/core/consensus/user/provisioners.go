package user

import (
	"fmt"
	"unsafe"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/sortedset"
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
	Provisioners struct {
		set     sortedset.Set
		members map[string]*Member
	}
)

func NewProvisioners() *Provisioners {
	return &Provisioners{
		set:     sortedset.New(),
		members: make(map[string]*Member),
	}
}

func (p *Provisioners) Size() int {
	return len(p.set)
}

//strPk is an efficient way to turn []byte into string
func strPk(pk []byte) string {
	return *(*string)(unsafe.Pointer(&pk))
}

func (p *Provisioners) MemberAt(i int) *Member {
	bigI := p.set[i]
	return p.members[strPk(bigI.Bytes())]
}

// GetMemberBLS returns a member of the provisioners from its BLS key
func (p *Provisioners) GetMember(pubKeyBLS []byte) *Member {
	return p.members[strPk(pubKeyBLS)]
}

// AddMember will add a Member to the Provisioners by using the bytes of a BLS public key. Returns the index at which the key has been inserted. If the member alredy exists, AddMember overrides its stake with the new one
func (p *Provisioners) AddMember(pubKeyEd, pubKeyBLS []byte, stake uint64) error {
	if len(pubKeyEd) != 32 {
		return fmt.Errorf("public key is %v bytes long instead of 32", len(pubKeyEd))
	}

	if len(pubKeyBLS) != 129 {
		return fmt.Errorf("public key is %v bytes long instead of 129", len(pubKeyBLS))
	}

	m := &Member{}
	m.PublicKeyEd = ed25519.PublicKey(pubKeyEd)

	pubKey := &bls.PublicKey{}
	if err := pubKey.Unmarshal(pubKeyBLS); err != nil {
		return err
	}

	m.PublicKeyBLS = *pubKey
	m.Stake = stake

	i := strPk(pubKeyBLS)
	// Check for duplicates
	inserted := p.set.Insert(pubKeyBLS)
	if !inserted {
		p.members[i].Stake = stake
		return nil
	}

	p.members[i] = m
	return nil
}

// RemoveMember will iterate over the committee and remove the specified Member.
func (p *Provisioners) Remove(pubKeyBLS []byte) bool {
	delete(p.members, strPk(pubKeyBLS))
	return p.set.Remove(pubKeyBLS)
}

// GetStake will find a certain provisioner in the committee by BLS public key,
// and return their stake.
func (p *Provisioners) GetStake(pubKeyBLS []byte) (uint64, error) {
	if len(pubKeyBLS) != 129 {
		return 0, fmt.Errorf("public key is %v bytes long instead of 129", len(pubKeyBLS))
	}

	i := strPk(pubKeyBLS)
	m, found := p.members[i]
	if !found {
		return 0, fmt.Errorf("public %v not found among provisioner set", pubKeyBLS)
	}

	return m.Stake, nil
}
