package user

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"sort"
)

// Member represents the bytes of a provisioner node's Ed25519 public key.
type Member [32]byte

// Equals will check if the two passed Members are of the same value.
func (m Member) Equals(member Member) bool {
	return bytes.Equal(m[:], member[:])
}

// String returns the hexadecimal string representation of a Member.
func (m Member) String() string {
	return hex.EncodeToString(m[:])
}

// Committee is a slice of Members, and makes up the current provisioner committee.
type Committee []Member

// AddMember will add a Member to the Committee by using the bytes of an Ed25519
// public key.
func (c *Committee) AddMember(pk []byte) error {
	if len(pk) != 32 {
		return fmt.Errorf("public key is %v bytes long instead of 32", len(pk))
	}

	var m Member
	copy(m[:], pk[:])

	// Check for duplicates
	for _, member := range *c {
		if m.Equals(member) {
			return nil
		}
	}

	*c = append(*c, m)
	return nil
}

// RemoveMember will iterate over the committee and remove the specified Member.
func (c *Committee) RemoveMember(pk []byte) error {
	if len(pk) != 32 {
		return fmt.Errorf("public key is %v bytes long instead of 32", len(pk))
	}

	var m Member
	copy(m[:], pk[:])

	for i, member := range *c {
		if m.Equals(member) {
			list := *c
			list = append(list[:i], list[i+1:]...)
			*c = list
		}
	}

	return nil
}

// Sort will sort the committee lexicographically
func (c *Committee) Sort() {
	list := *c
	sort.SliceStable(list, func(i, j int) bool {
		return list[i].String() < list[j].String()
	})
	*c = list
}
