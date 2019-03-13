package user

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"sort"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"golang.org/x/crypto/ed25519"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/hash"
)

// Member contains the bytes of a provisioner's Ed25519 public key,
// the bytes of his BLS public key, and how much he has staked.
type Member struct {
	PublicKeyEd  ed25519.PublicKey
	PublicKeyBLS bls.PublicKey
	Stake        uint64
}

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

// Provisioners is a slice of Members, and makes up the current provisioner committee.
type Provisioners []Member

// AddMember will add a Member to the Provisioners by using the bytes of an Ed25519
// public key.
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
func (p *Provisioners) RemoveMember(pubKeyEd []byte) error {
	if len(pubKeyEd) != 32 {
		return fmt.Errorf("public key is %v bytes long instead of 32", len(pubKeyEd))
	}

	var m Member
	m.PublicKeyEd = ed25519.PublicKey(pubKeyEd)

	for i, member := range *p {
		if m.EdEquals(member) {
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

// CreateVotingCommittee will run the deterministic sortition function, which determines
// who will be in the committee for a given step.
func (p Provisioners) CreateVotingCommittee(round, totalWeight uint64,
	step uint8) (map[string]uint8, error) {

	votingCommittee := make(map[string]uint8)
	W := new(big.Int).SetUint64(totalWeight)

	for i := 0; i < len(p); i++ {
		// Create message to hash
		msg := createSortitionMessage(round, step, uint8(i))

		// Hash message
		hash, err := hash.Sha3256(msg)
		if err != nil {
			return nil, err
		}

		// Get a committee member
		member, err := p.getCommitteeMember(hash, W)
		if err != nil {
			return nil, err
		}

		votingCommittee[member]++
	}

	return votingCommittee, nil
}

// createSortitionMessage will create the slice of bytes containing all of the information
// we will hash during sortition, and return it.
func createSortitionMessage(round uint64, step uint8, i uint8) []byte {
	msg := make([]byte, 12)
	binary.LittleEndian.PutUint64(msg[:8], round)
	msg = append(msg, byte(step))
	msg = append(msg, byte(i))

	return msg
}

// getCommitteeMember generates a score from the given hash, and then
// walks through the committee set, while deducting each node's stake from the score
// until we reach zero. The public key of the node that the function ends on
// will be returned as a hexadecimal string.
func (p Provisioners) getCommitteeMember(hash []byte, W *big.Int) (string, error) {

	// Generate score
	hashNum := new(big.Int).SetBytes(hash)
	score := new(big.Int).Mod(hashNum, W).Uint64()

	// Walk through the committee set and keep deducting until we reach zero
	for _, member := range p {
		if member.Stake >= score {
			return member.BLSString(), nil
		}

		score -= member.Stake
	}

	// With the Committee and TotalWeight fields properly set,
	// we should never end up here. However, if we do, return an error.
	return "", errors.New("committee and total weight values do not correspond")
}
