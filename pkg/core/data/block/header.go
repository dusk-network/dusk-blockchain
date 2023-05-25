// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package block

import (
	"bytes"
	"encoding/binary"

	"github.com/dusk-network/dusk-crypto/hash"
)

const (
	// HeaderHashSize size of a block header hash in bytes.
	HeaderHashSize = 32
	// HeightSize size of a block height field in bytes.
	HeightSize = 8
)

// EmptyHash ...
var EmptyHash [32]byte

// Header defines a block header on a Dusk block.
type Header struct {
	Version   uint8  `json:"version"`   // Block version byte
	Height    uint64 `json:"height"`    // Block height
	Timestamp int64  `json:"timestamp"` // Block timestamp
	GasLimit  uint64 `json:"gaslimit"`  // Block gas limit
	Iteration uint8  `json:"iteration"` // Iteration at which block is produced

	PrevBlockHash      []byte `json:"prev-hash"`  // Hash of previous block (32 bytes)
	Seed               []byte `json:"seed"`       // Marshaled BLS signature or hash of the previous block seed (32 bytes) //TODO-1508: sig or hash?
	GeneratorBlsPubkey []byte `json:"generator"`  // Generator BLS Public Key (96 bytes)
	TxRoot             []byte `json:"tx-root"`    // Root hash of the merkle tree containing all txes (32 bytes)
	StateHash          []byte `json:"state-hash"` // Root hash of the Rusk Contract Storage state

	Hash []byte `json:"hash"` // Hash of all previous fields

	*Certificate `json:"certificate"` // Block certificate
}

// NewHeader creates a new Block Header zero-ed.
func NewHeader() *Header {
	return &Header{
		Hash:               EmptyHash[:],
		PrevBlockHash:      EmptyHash[:],
		Seed:               EmptyHash[:],
		StateHash:          EmptyHash[:],
		GeneratorBlsPubkey: make([]byte, 96),
		Certificate:        EmptyCertificate(),
		GasLimit:           0,
	}
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (b *Header) Copy() *Header {
	h := &Header{
		Certificate: b.Certificate.Copy(),
		Version:     b.Version,
		Height:      b.Height,
		Timestamp:   b.Timestamp,
		GasLimit:    b.GasLimit,
		Iteration:   b.Iteration,
	}

	h.PrevBlockHash = make([]byte, len(b.PrevBlockHash))
	copy(h.PrevBlockHash, b.PrevBlockHash)
	h.Seed = make([]byte, len(b.Seed))
	copy(h.Seed, b.Seed)
	h.GeneratorBlsPubkey = make([]byte, len(b.GeneratorBlsPubkey))
	copy(h.GeneratorBlsPubkey, b.GeneratorBlsPubkey)
	h.TxRoot = make([]byte, len(b.TxRoot))
	copy(h.TxRoot, b.TxRoot)
	h.Hash = make([]byte, len(b.Hash))
	copy(h.Hash, b.Hash)
	h.StateHash = make([]byte, len(b.StateHash))
	copy(h.StateHash, b.StateHash)

	return h
}

// CalculateHash will calculate and return this block header's hash by encoding all the relevant
// fields and then hashing the result.
func (b *Header) CalculateHash() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := marshalHashable(buf, b); err != nil {
		return nil, err
	}

	return hash.Sha3256(buf.Bytes())
}

// Equals returns true if headers are equal.
func (b *Header) Equals(other *Header) bool {
	if other == nil {
		return false
	}

	if b.Version != other.Version {
		return false
	}

	if b.Timestamp != other.Timestamp {
		return false
	}

	if b.GasLimit != other.GasLimit {
		return false
	}

	if !bytes.Equal(b.PrevBlockHash, other.PrevBlockHash) {
		return false
	}

	if !bytes.Equal(b.Seed, other.Seed) {
		return false
	}

	if !bytes.Equal(b.TxRoot, other.TxRoot) {
		return false
	}

	if !bytes.Equal(b.GeneratorBlsPubkey, other.GeneratorBlsPubkey) {
		return false
	}

	if !b.Certificate.Equals(other.Certificate) {
		return false
	}

	if !bytes.Equal(b.StateHash, other.StateHash) {
		return false
	}

	if !bytes.Equal(b.Hash, other.Hash) {
		return false
	}

	if b.Iteration != other.Iteration {
		return false
	}

	return true
}

func marshalHashable(b *bytes.Buffer, h *Header) error {
	if err := binary.Write(b, binary.LittleEndian, h.Version); err != nil {
		return err
	}

	if err := binary.Write(b, binary.LittleEndian, h.Height); err != nil {
		return err
	}

	if err := binary.Write(b, binary.LittleEndian, uint64(h.Timestamp)); err != nil {
		return err
	}

	if err := binary.Write(b, binary.BigEndian, h.PrevBlockHash); err != nil {
		return err
	}

	if err := binary.Write(b, binary.BigEndian, h.Seed); err != nil {
		return err
	}

	if err := binary.Write(b, binary.BigEndian, h.StateHash); err != nil {
		return err
	}

	if err := binary.Write(b, binary.BigEndian, h.GeneratorBlsPubkey); err != nil {
		return err
	}

	if err := binary.Write(b, binary.BigEndian, h.TxRoot); err != nil {
		return err
	}

	if err := binary.Write(b, binary.LittleEndian, h.GasLimit); err != nil {
		return err
	}

	if err := b.WriteByte(h.Iteration); err != nil {
		return err
	}

	return nil
}
