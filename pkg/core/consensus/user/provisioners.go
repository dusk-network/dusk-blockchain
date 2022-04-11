// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package user

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/dusk-network/bls12_381-sign/go/cgo/bls"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/sortedset"
)

type (
	// Member contains the bytes of a provisioner's Ed25519 public key,
	// the bytes of his BLS public key, and how much he has staked.
	Member struct {
		PublicKeyBLS    []byte  `json:"bls_key"`
		RawPublicKeyBLS []byte  `json:"raw_bls_key"`
		Stakes          []Stake `json:"stakes"`
	}

	// Provisioners is a map of Members, and makes up the current set of provisioners.
	Provisioners struct {
		Set     sortedset.Set      `json:"set"`
		Members map[string]*Member `json:"members"`
	}

	// Stake represents the Provisioner's stake.
	Stake struct {
		Value       uint64 `json:"value"`
		Reward      uint64 `json:"reward"`
		Counter     uint64 `json:"counter"`
		Eligibility uint64 `json:"eligibility"`
	}
)

// Copy deeply a set of Provisioners.
func (p Provisioners) Copy() Provisioners {
	cpy := Provisioners{
		Set:     p.Set.Copy(),
		Members: make(map[string]*Member),
	}

	for k, v := range p.Members {
		cpy.Members[k] = v.Copy()
	}

	return cpy
}

// AddStake appends a stake to the stake set.
func (m *Member) AddStake(stake Stake) {
	m.Stakes = append(m.Stakes, stake)
}

// Copy deep a Member.
func (m *Member) Copy() *Member {
	cpy := &Member{
		PublicKeyBLS:    make([]byte, len(m.PublicKeyBLS)),
		RawPublicKeyBLS: make([]byte, len(m.RawPublicKeyBLS)),
		Stakes:          make([]Stake, len(m.Stakes)),
	}

	copy(cpy.PublicKeyBLS, m.PublicKeyBLS)
	copy(cpy.RawPublicKeyBLS, m.RawPublicKeyBLS)

	for i, s := range m.Stakes {
		cpy.Stakes[i] = Stake{
			Value:       s.Value,
			Counter:     s.Counter,
			Reward:      s.Reward,
			Eligibility: s.Eligibility,
		}
	}

	return cpy
}

// RemoveStake removes a Stake (most likely because it expired).
// Note: At the moment there's a 1-to-1 relationship between provisioner
// and stake. In the future this could potentially change.
// See also github.com/dusk-network/rusk/issues/579.
func (m *Member) RemoveStake(idx int) {
	m.Stakes[idx] = m.Stakes[len(m.Stakes)-1]
	m.Stakes = m.Stakes[:len(m.Stakes)-1]
}

// SubtractFromStake detracts an amount `value` from the Stake of a Provisioner.
func (m *Member) SubtractFromStake(value uint64) uint64 {
	for i := 0; i < len(m.Stakes); i++ {
		if m.Stakes[i].Value > 0 {
			if m.Stakes[i].Value < value {
				subtracted := m.Stakes[i].Value
				m.Stakes[i].Value = 0

				return subtracted
			}

			m.Stakes[i].Value -= value
			return value
		}
	}

	return 0
}

// NewProvisioners instantiates the Provisioners sortedset of members.
func NewProvisioners() *Provisioners {
	return &Provisioners{
		Set:     sortedset.New(),
		Members: make(map[string]*Member),
	}
}

// Add a Member to the Provisioners by using the bytes of a BLS public key.
func (p *Provisioners) Add(pubKeyBLS []byte, value, reward, counter, eligibility uint64) error {
	if len(pubKeyBLS) != 96 {
		return fmt.Errorf("public key is %v bytes long instead of 96", len(pubKeyBLS))
	}

	i := string(pubKeyBLS)
	stake := Stake{Value: value, Reward: reward, Counter: counter, Eligibility: eligibility}

	// Check for duplicates
	_, inserted := p.Set.IndexOf(pubKeyBLS)
	if inserted {
		// If they already exist, just add their new stake
		p.Members[i].AddStake(stake)
		return nil
	}

	// This is a new provisioner, so let's initialize the Member struct and add them to the list
	p.Set.Insert(pubKeyBLS)

	var rpk []byte
	var err error

	if rpk, err = bls.PkToRaw(pubKeyBLS); err != nil {
		return err
	}

	m := &Member{
		PublicKeyBLS:    pubKeyBLS,
		RawPublicKeyBLS: rpk,
	}

	m.AddStake(stake)

	p.Members[i] = m
	return nil
}

// SubsetSizeAt returns how many provisioners are active on a given round.
// This function is used to determine the correct committee size for
// sortition in the case where one or more provisioner stakes have not
// yet become active, or have just expired. Note that this function will
// only give an accurate result if the round given is either identical
// or close to the current block height, as stakes are removed soon
// after they expire.
func (p Provisioners) SubsetSizeAt(round uint64) int {
	var size int

	for _, member := range p.Members {
		for _, stake := range member.Stakes {
			if round >= stake.Eligibility {
				size++
				break
			}
		}
	}

	return size
}

// MemberAt returns the Member at a certain index.
func (p Provisioners) MemberAt(i int) (*Member, error) {
	if i > len(p.Set)-1 {
		return nil, errors.New("index out of bound")
	}

	bigI := p.Set[i]
	return p.Members[string(bigI.Bytes())], nil
}

// GetMember returns a member of the provisioners from its BLS public key.
func (p Provisioners) GetMember(pubKeyBLS []byte) *Member {
	return p.Members[string(pubKeyBLS)]
}

// GetStake will find a certain provisioner in the committee by BLS public key,
// and return their stake.
func (p Provisioners) GetStake(pubKeyBLS []byte) (uint64, error) {
	if len(pubKeyBLS) != 96 {
		return 0, fmt.Errorf("public key is %v bytes long instead of 129", len(pubKeyBLS))
	}

	m, found := p.Members[string(pubKeyBLS)]
	if !found {
		return 0, fmt.Errorf("public key %v not found among provisioner set", pubKeyBLS)
	}

	var totalStake uint64
	for _, stake := range m.Stakes {
		totalStake += stake.Value
	}

	return totalStake, nil
}

// TotalWeight is the sum of all stakes of the provisioners.
func (p *Provisioners) TotalWeight() (totalWeight uint64) {
	for _, member := range p.Members {
		for _, stake := range member.Stakes {
			totalWeight += stake.Value
		}
	}

	return totalWeight
}

// GetRawPublicKeyBLS returns a member uncompressed BLS public key.
// Returns nil if member not found.
func (p Provisioners) GetRawPublicKeyBLS(pubKeyBLS []byte) []byte {
	m, ok := p.Members[string(pubKeyBLS)]
	if !ok {
		return nil
	}

	return m.RawPublicKeyBLS
}

// MarshalProvisioners ...
func MarshalProvisioners(r *bytes.Buffer, p *Provisioners) error {
	if err := encoding.WriteVarInt(r, uint64(len(p.Members))); err != nil {
		return err
	}

	for _, member := range p.Members {
		if err := marshalMember(r, *member); err != nil {
			return err
		}
	}

	return nil
}

func marshalMember(r *bytes.Buffer, member Member) error {
	if err := encoding.WriteVarBytes(r, member.PublicKeyBLS); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, member.RawPublicKeyBLS); err != nil {
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
	if err := encoding.WriteUint64LE(r, stake.Value); err != nil {
		return err
	}

	if err := encoding.WriteUint64LE(r, stake.Reward); err != nil {
		return err
	}
	if err := encoding.WriteUint64LE(r, stake.Counter); err != nil {
		return err
	}

	if err := encoding.WriteUint64LE(r, stake.Eligibility); err != nil {
		return err
	}

	return nil
}

// UnmarshalProvisioners unmarshal provisioner set from a buffer.
func UnmarshalProvisioners(r *bytes.Buffer) (Provisioners, error) {
	lMembers, err := encoding.ReadVarInt(r)
	if err != nil {
		return Provisioners{}, err
	}

	members := make([]*Member, lMembers)
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
		memberMap[string(member.PublicKeyBLS)] = member
	}

	return Provisioners{
		Set:     set,
		Members: memberMap,
	}, nil
}

func unmarshalMember(r *bytes.Buffer) (*Member, error) {
	member := &Member{}
	if err := encoding.ReadVarBytes(r, &member.PublicKeyBLS); err != nil {
		return nil, err
	}

	if err := encoding.ReadVarBytes(r, &member.RawPublicKeyBLS); err != nil {
		return nil, err
	}

	lStakes, err := encoding.ReadVarInt(r)
	if err != nil {
		return nil, err
	}

	member.Stakes = make([]Stake, lStakes)
	for i := uint64(0); i < lStakes; i++ {
		member.Stakes[i], err = unmarshalStake(r)
		if err != nil {
			return nil, err
		}
	}

	return member, nil
}

func unmarshalStake(r *bytes.Buffer) (Stake, error) {
	stake := Stake{}
	if err := encoding.ReadUint64LE(r, &stake.Value); err != nil {
		return Stake{}, err
	}

	if err := encoding.ReadUint64LE(r, &stake.Reward); err != nil {
		return Stake{}, err
	}

	if err := encoding.ReadUint64LE(r, &stake.Counter); err != nil {
		return Stake{}, err
	}

	if err := encoding.ReadUint64LE(r, &stake.Eligibility); err != nil {
		return Stake{}, err
	}

	return stake, nil
}

// Format implements fmt.Formatter interface.
func (s Stake) Format(f fmt.State, c rune) {
	r := fmt.Sprintf("Value: %d, Reward: %d, Counter: %d, Eligibility: %d", s.Value, s.Reward, s.Counter, s.Eligibility)
	_, _ = f.Write([]byte(r))
}
