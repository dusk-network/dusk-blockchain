// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package header

// TODO: consider moving this into the message package

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/dusk-network/bls12_381-sign/go/cgo/bls"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message/payload"
	"github.com/dusk-network/dusk-blockchain/pkg/util"
	crypto "github.com/dusk-network/dusk-crypto/hash"
)

type (
	// Header is an embeddable struct representing the consensus event header fields.
	Header struct {
		PubKeyBLS []byte
		Round     uint64
		Step      uint8
		BlockHash []byte
	}
)

// Writer writes the header to a Buffer. It is an interface injected into components.
type Writer interface {
	WriteHeader([]byte, *bytes.Buffer) error
}

// Phase is used to introduce a time order to the Header.
type Phase uint8

const (
	// Same indicates that headers belong to the same phase.
	Same Phase = iota
	// Before indicates that the header indicates a past event.
	Before
	// After indicates that the header indicates a future event.
	After
)

// New will create a Header instance.
func New() Header {
	return Header{
		PubKeyBLS: make([]byte, 96),
		Round:     uint64(0),
		Step:      uint8(0),
		BlockHash: make([]byte, 32),
	}
}

// Copy complies with the Safe interface.
func (h Header) Copy() payload.Safe {
	hdr := Header{
		Round: h.Round,
		Step:  h.Step,
	}

	if h.BlockHash != nil {
		hdr.BlockHash = make([]byte, len(h.BlockHash))
		copy(hdr.BlockHash, h.BlockHash)
	}

	if h.PubKeyBLS != nil {
		hdr.PubKeyBLS = make([]byte, len(h.PubKeyBLS))
		copy(hdr.PubKeyBLS, h.PubKeyBLS)
	}

	return hdr
}

// State returns the Header struct itself. It is mandate by the
// consensus.InternalPacket interface.
func (h Header) State() Header {
	return h
}

// Sender implements wire.Event.
// Returns the BLS public key of the event sender.
// It is part of the consensus.Packet interface.
func (h Header) Sender() []byte {
	return h.PubKeyBLS
}

// CompareRoundAndStep Compare headers to establish time order.
func (h Header) CompareRoundAndStep(round uint64, step uint8) Phase {
	comparison := h.CompareRound(round)
	if comparison == Same {
		if h.Step < step {
			return Before
		}

		if h.Step == step {
			return Same
		}

		return After
	}

	return comparison
}

// CompareRound compares the actual h Header v @round returning the respective phase.
func (h Header) CompareRound(round uint64) Phase {
	if h.Round < round {
		return Before
	}

	if h.Round == round {
		return Same
	}

	return After
}

func (h Header) String() string {
	var sb strings.Builder

	_, _ = sb.WriteString(fmt.Sprintf("round='%d' step='%d'", h.Round, h.Step))
	_, _ = sb.WriteString(" sender='")
	_, _ = sb.WriteString(util.StringifyBytes(h.PubKeyBLS))
	_, _ = sb.WriteString("' block_hash='")
	_, _ = sb.WriteString(util.StringifyBytes(h.BlockHash))
	_, _ = sb.WriteString("'")

	return sb.String()
}

// Equal implements wire.Event.
// Checks if two headers are the same.
func (h Header) Equal(e wire.Event) bool {
	other, ok := e.(Header)
	return ok && (bytes.Equal(h.PubKeyBLS, other.PubKeyBLS)) &&
		(h.Round == other.Round) && (h.Step == other.Step) &&
		(bytes.Equal(h.BlockHash, other.BlockHash))
}

// Marshal a Header into a Buffer.
func Marshal(r *bytes.Buffer, ev Header) error {
	if err := encoding.WriteVarBytes(r, ev.PubKeyBLS); err != nil {
		return err
	}

	return MarshalFields(r, ev)
}

// Compose is useful when header information is cached and there is an opportunity to avoid unnecessary allocations.
func Compose(blsPubKey bytes.Buffer, phase bytes.Buffer, hash []byte) (bytes.Buffer, error) {
	if _, err := blsPubKey.ReadFrom(&phase); err != nil {
		return bytes.Buffer{}, err
	}

	if err := encoding.Write256(&blsPubKey, hash); err != nil {
		return bytes.Buffer{}, err
	}

	return blsPubKey, nil
}

// Unmarshal unmarshals the buffer into a Header.
func Unmarshal(r *bytes.Buffer, ev *Header) error {
	// Decoding PubKey BLS
	if err := encoding.ReadVarBytes(r, &ev.PubKeyBLS); err != nil {
		return err
	}

	return UnmarshalFields(r, ev)
}

// MarshalFields marshals the core field of the Header (i.e. Round, Step and BlockHash).
func MarshalFields(r *bytes.Buffer, h Header) error {
	if err := encoding.WriteUint64LE(r, h.Round); err != nil {
		return err
	}

	if err := encoding.WriteUint8(r, h.Step); err != nil {
		return err
	}

	return encoding.Write256(r, h.BlockHash)
}

// UnmarshalFields unmarshals the core field of the Header (i.e. Round, Step and BlockHash).
func UnmarshalFields(r *bytes.Buffer, h *Header) error {
	if err := encoding.ReadUint64LE(r, &h.Round); err != nil {
		return err
	}

	if err := encoding.ReadUint8(r, &h.Step); err != nil {
		return err
	}

	h.BlockHash = make([]byte, 32)
	return encoding.Read256(r, h.BlockHash)
}

// MarshalSignableVote marshals the fields necessary for a Committee member to cast
// a Vote (namely the Round, the Step and the BlockHash).
// Note: UnmarshalSignableVote does not make sense as the only reason to use it would be if we could somehow revert a signature to the preimage and thus unmarshal it into a struct :P.
func MarshalSignableVote(r *bytes.Buffer, h Header) error {
	return MarshalFields(r, h)
}

// VerifySignatures verifies the BLS aggregated signature carried by consensus related messages.
// The signed message needs to carry information about the round, the step and the blockhash.
func VerifySignatures(round uint64, step uint8, blockHash, apk, sig []byte) error {
	signed := new(bytes.Buffer)
	vote := Header{
		Round:     round,
		Step:      step,
		BlockHash: blockHash,
	}

	if err := MarshalSignableVote(signed, vote); err != nil {
		return err
	}

	return bls.Verify(apk, sig, signed.Bytes())
}

// Mock a Header.
func Mock() Header {
	hash, _ := crypto.RandEntropy(32)
	k := key.NewRandKeys()
	pubkey := k.BLSPubKey
	buf := make([]byte, 8)
	_, _ = rand.Read(buf)
	round := binary.LittleEndian.Uint64(buf)

	step := []byte{0}
	_, _ = rand.Reader.Read(step)

	return Header{
		BlockHash: hash,
		Round:     round,
		Step:      step[0],
		PubKeyBLS: pubkey,
	}
}
