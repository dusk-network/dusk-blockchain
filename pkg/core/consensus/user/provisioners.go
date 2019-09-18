package user

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/sortedset"
)

type (
	// Member contains the bytes of a provisioner's Ed25519 public key,
	// the bytes of his BLS public key, and how much he has staked.
	Member struct {
		PublicKeyEd  []byte
		PublicKeyBLS []byte
		Stakes       []Stake
	}

	// Provisioners is a map of Members, and makes up the current set of provisioners.
	Provisioners struct {
		Set     sortedset.Set
		Members map[string]*Member
	}

	Stake struct {
		Amount    uint64
		EndHeight uint64
	}
)

func (m *Member) AddStake(stake Stake) {
	m.Stakes = append(m.Stakes, stake)
}

func (m *Member) RemoveStake(idx int) {
	m.Stakes[idx] = m.Stakes[len(m.Stakes)-1]
	m.Stakes = m.Stakes[:len(m.Stakes)-1]
}

func NewProvisioners() *Provisioners {
	return &Provisioners{
		Set:     sortedset.New(),
		Members: make(map[string]*Member),
	}
}

// MemberAt returns the Member at a certain index.
func (p Provisioners) MemberAt(i int) *Member {
	bigI := p.Set[i]
	return p.Members[string(bigI.Bytes())]
}

// GetMember returns a member of the provisioners from its BLS public key.
func (p Provisioners) GetMember(pubKeyBLS []byte) *Member {
	return p.Members[string(pubKeyBLS)]
}

// GetStake will find a certain provisioner in the committee by BLS public key,
// and return their stake.
func (p Provisioners) GetStake(pubKeyBLS []byte) (uint64, error) {
	if len(pubKeyBLS) != 129 {
		return 0, fmt.Errorf("public key is %v bytes long instead of 129", len(pubKeyBLS))
	}

	i := string(pubKeyBLS)
	m, found := p.Members[i]
	if !found {
		return 0, fmt.Errorf("public key %v not found among provisioner set", pubKeyBLS)
	}

	var totalStake uint64
	for _, stake := range m.Stakes {
		totalStake += stake.Amount
	}

	return totalStake, nil
}

func (p *Provisioners) TotalWeight() (totalWeight uint64) {
	for _, member := range p.Members {
		for _, stake := range member.Stakes {
			totalWeight += stake.Amount
		}
	}

	return totalWeight
}

func MarshalMembers(r *bytes.Buffer, members []Member) error {
	if err := encoding.WriteVarInt(r, uint64(len(members))); err != nil {
		return err
	}

	for _, member := range members {
		if err := marshalMember(r, member); err != nil {
			return err
		}
	}

	return nil
}

func marshalMember(r *bytes.Buffer, member Member) error {
	if err := encoding.Write256(r, member.PublicKeyEd); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, member.PublicKeyBLS); err != nil {
		return err
	}

	if err := encoding.WriteVarInt(r, uint64(len(member.Stakes))); err != nil {
		return err
	}

	for _, stake := range member.Stakes {
		if err := marshalStake(r, stake); err != nil {
			return err
		}
	}

	return nil
}

func marshalStake(r *bytes.Buffer, stake Stake) error {
	if err := encoding.WriteUint64(r, binary.LittleEndian, stake.Amount); err != nil {
		return err
	}

	if err := encoding.WriteUint64(r, binary.LittleEndian, stake.EndHeight); err != nil {
		return err
	}

	return nil
}

func UnmarshalProvisioners(r *bytes.Buffer) (Provisioners, error) {
	lMembers, err := encoding.ReadVarInt(r)
	if err != nil {
		return Provisioners{}, err
	}

	members := make([]Member, lMembers)
	for i := uint64(0); i < lMembers; i++ {
		members[i], err = unmarshalMember(r)
		if err != nil {
			return Provisioners{}, err
		}
	}

	// Reconstruct sorted set and member map
	set := sortedset.New()
	memberMap := make(map[string]*Member)
	for _, member := range members {
		set.Insert(member.PublicKeyBLS)
		memberMap[string(member.PublicKeyBLS)] = &member
	}

	return Provisioners{
		Set:     set,
		Members: memberMap,
	}, nil
}

func unmarshalMember(r *bytes.Buffer) (Member, error) {
	member := Member{}
	if err := encoding.Read256(r, &member.PublicKeyEd); err != nil {
		return Member{}, err
	}

	if err := encoding.ReadVarBytes(r, &member.PublicKeyBLS); err != nil {
		return Member{}, err
	}

	lStakes, err := encoding.ReadVarInt(r)
	if err != nil {
		return Member{}, err
	}

	member.Stakes = make([]Stake, lStakes)
	for i := uint64(0); i < lStakes; i++ {
		member.Stakes[i], err = unmarshalStake(r)
		if err != nil {
			return Member{}, err
		}
	}

	return member, nil
}

func unmarshalStake(r *bytes.Buffer) (Stake, error) {
	stake := Stake{}
	if err := encoding.ReadUint64(r, binary.LittleEndian, &stake.Amount); err != nil {
		return Stake{}, err
	}

	if err := encoding.ReadUint64(r, binary.LittleEndian, &stake.EndHeight); err != nil {
		return Stake{}, err
	}

	return stake, nil
}
