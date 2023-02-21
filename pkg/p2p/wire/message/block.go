// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package message

import (
	"bytes"
	"errors"
	"math"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
)

// MarshalBlock marshals a block into a binary buffer.
func MarshalBlock(r *bytes.Buffer, b *block.Block) error {
	if err := MarshalHeader(r, b.Header); err != nil {
		return err
	}

	lenTxs := uint64(len(b.Txs))
	if err := encoding.WriteVarInt(r, lenTxs); err != nil {
		return err
	}

	// TODO: parallelize transaction serialization
	for _, tx := range b.Txs {
		if err := transactions.Marshal(r, tx); err != nil {
			return err
		}
	}

	return nil
}

// UnmarshalBlockMessage unmarshals a block from a binary buffer into
// a SerializableMessage.
func UnmarshalBlockMessage(r *bytes.Buffer, m SerializableMessage) error {
	blk := block.NewBlock()
	if err := UnmarshalBlock(r, blk); err != nil {
		return err
	}

	m.SetPayload(*blk)
	return nil
}

// UnmarshalBlock unmarshals a block from a binary buffer.
func UnmarshalBlock(r *bytes.Buffer, b *block.Block) error {
	if err := UnmarshalHeader(r, b.Header); err != nil {
		return err
	}

	lTxs, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
	}

	// Maximum amount of transactions we can decode at once is
	// math.MaxInt32 / 8, since they are pointers (uint64)
	if lTxs > (math.MaxInt32 / 8) {
		return errors.New("block tx count too large")
	}

	b.Txs = make([]transactions.ContractCall, lTxs)
	for i := range b.Txs {
		c := transactions.NewTransaction()
		if err := transactions.Unmarshal(r, c); err != nil {
			return err
		}

		b.Txs[i] = c
	}

	return nil
}

// MarshalHashable marshals the hashable part of the block into a binary buffer.
func MarshalHashable(r *bytes.Buffer, h *block.Header) error {
	if err := encoding.WriteUint8(r, h.Version); err != nil {
		return err
	}

	if err := encoding.WriteUint64LE(r, h.Height); err != nil {
		return err
	}

	if err := encoding.WriteUint64LE(r, uint64(h.Timestamp)); err != nil {
		return err
	}

	if err := encoding.Write256(r, h.PrevBlockHash); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, h.Seed); err != nil {
		return err
	}

	if err := encoding.Write256(r, h.StateHash); err != nil {
		return err
	}

	if err := encoding.WriteBLSPKey(r, h.GeneratorBlsPubkey); err != nil {
		return err
	}

	if err := encoding.WriteUint64LE(r, h.GasLimit); err != nil {
		return err
	}

	if err := encoding.WriteUint8(r, h.Iteration); err != nil {
		return err
	}

	return nil
}

// MarshalHeader marshals the header of a block into a binary buffer.
func MarshalHeader(r *bytes.Buffer, h *block.Header) error {
	if err := MarshalHashable(r, h); err != nil {
		return err
	}

	if err := MarshalCertificate(r, h.Certificate); err != nil {
		return err
	}

	if err := encoding.Write256(r, h.Hash); err != nil {
		return err
	}

	return nil
}

// UnmarshalHeader unmarshal a block header from a binary buffer.
func UnmarshalHeader(r *bytes.Buffer, h *block.Header) error {
	if err := encoding.ReadUint8(r, &h.Version); err != nil {
		return err
	}

	if err := encoding.ReadUint64LE(r, &h.Height); err != nil {
		return err
	}

	var timestamp uint64
	if err := encoding.ReadUint64LE(r, &timestamp); err != nil {
		return err
	}

	h.Timestamp = int64(timestamp)

	h.PrevBlockHash = make([]byte, 32)
	if err := encoding.Read256(r, h.PrevBlockHash); err != nil {
		return err
	}

	h.Seed = make([]byte, 0)
	if err := encoding.ReadVarBytes(r, &h.Seed); err != nil {
		return err
	}

	h.StateHash = make([]byte, 32)
	if err := encoding.Read256(r, h.StateHash); err != nil {
		return err
	}

	h.GeneratorBlsPubkey = make([]byte, 96)
	if err := encoding.ReadBLSPKey(r, h.GeneratorBlsPubkey); err != nil {
		return err
	}

	if err := encoding.ReadUint64LE(r, &h.GasLimit); err != nil {
		return err
	}

	if err := encoding.ReadUint8(r, &h.Iteration); err != nil {
		return err
	}

	if err := UnmarshalCertificate(r, h.Certificate); err != nil {
		return err
	}

	h.Hash = make([]byte, 32)
	if err := encoding.Read256(r, h.Hash); err != nil {
		return err
	}

	return nil
}

// MarshalCertificate marshals a certificate.
func MarshalCertificate(r *bytes.Buffer, c *block.Certificate) error {
	if err := encoding.WriteVarBytes(r, c.StepOneBatchedSig); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, c.StepTwoBatchedSig); err != nil {
		return err
	}

	if err := encoding.WriteUint8(r, c.Step); err != nil {
		return err
	}

	if err := encoding.WriteUint64LE(r, c.StepOneCommittee); err != nil {
		return err
	}

	if err := encoding.WriteUint64LE(r, c.StepTwoCommittee); err != nil {
		return err
	}

	return nil
}

// UnmarshalCertificate unmarshals a certificate.
func UnmarshalCertificate(r *bytes.Buffer, c *block.Certificate) error {
	c.StepOneBatchedSig = make([]byte, 0)
	if err := encoding.ReadVarBytes(r, &c.StepOneBatchedSig); err != nil {
		return err
	}

	c.StepTwoBatchedSig = make([]byte, 0)
	if err := encoding.ReadVarBytes(r, &c.StepTwoBatchedSig); err != nil {
		return err
	}

	if err := encoding.ReadUint8(r, &c.Step); err != nil {
		return err
	}

	if err := encoding.ReadUint64LE(r, &c.StepOneCommittee); err != nil {
		return err
	}

	if err := encoding.ReadUint64LE(r, &c.StepTwoCommittee); err != nil {
		return err
	}

	return nil
}
