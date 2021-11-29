// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package message

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message/payload"
	log "github.com/sirupsen/logrus"
)

// AggrAgreement stores an agreement (with the provisioners) and
// the bitset representing those who voted in favor.
type AggrAgreement struct {
	Agreement
	Bitset  uint64
	AggrSig []byte
}

// NewAggrAgreement creates an AggrAgreement message with the given fields.
func NewAggrAgreement(payload Agreement, bitset uint64, aggrSig []byte) AggrAgreement {
	return AggrAgreement{
		Agreement: payload,
		Bitset:    bitset,
		AggrSig:   aggrSig,
	}
}

// newAggrAgreement creates an empty AggrAgreement message.
func newAggrAgreement() *AggrAgreement {
	return &AggrAgreement{
		Agreement: *newAgreement(),
		Bitset:    uint64(0),
		AggrSig:   make([]byte, 48), // TODO: Use of hardcoded constant is discouraged
	}
}

// Copy deeply a AggrAgreement.
func (a AggrAgreement) Copy() payload.Safe {
	cpy := AggrAgreement{
		Agreement: a.Agreement.Copy().(Agreement),
		Bitset:    a.Bitset,
		AggrSig:   make([]byte, 48),
	}
	copy(cpy.AggrSig, a.AggrSig)
	return cpy
}

// String returns a string representation of a AggrAgreement message.
func (a AggrAgreement) String() string {
	var sb strings.Builder
	_, _ = sb.WriteString(fmt.Sprintf("Agreement: %s\n Bitset: %d\n AggrSig: %v\n", a.Agreement, a.Bitset, a.AggrSig))
	return sb.String()
}

// MarshalAggrAgreement marshal the AggrAgreement into a Buffer.
func MarshalAggrAgreement(r *bytes.Buffer, aggr AggrAgreement) error {
	// Payload
	if err := MarshalAgreement(r, aggr.Agreement); err != nil {
		log.WithError(err).Errorln("failed to marshal agreement (from aggr agreement)")
		return err
	}
	// Bitset
	if err := encoding.WriteUint64LE(r, aggr.Bitset); err != nil {
		log.WithError(err).Errorln("failed to marshal bitset")
		return err
	}

	// AggrSig
	if err := encoding.WriteVarBytes(r, aggr.AggrSig); err != nil {
		log.WithError(err).Errorln("failed to marshal aggr signature")
		return err
	}

	return nil
}

// UnmarshalAggrAgreement unmarshals the buffer into an AggrAgreement.
func UnmarshalAggrAgreement(r *bytes.Buffer, a *AggrAgreement) error {
	if err := header.Unmarshal(r, &a.Agreement.hdr); err != nil {
		return err
	}

	if err := UnmarshalAgreement(r, &a.Agreement); err != nil {
		log.WithError(err).Errorln("failed to unmarshal agreement (from aggr agreement)")
		return err
	}

	// Bitset
	if err := encoding.ReadUint64LE(r, &a.Bitset); err != nil {
		log.WithError(err).Errorln("failed to unmarshal bitset")
		return err
	}

	// AggrSig
	if err := encoding.ReadVarBytes(r, &a.AggrSig); err != nil {
		log.WithError(err).Errorln("failed to unmarshal signature")
		return err
	}

	return nil
}

// UnmarshalAggrAgreementMessage unmarshal a network inbound AggrAgreement.
func UnmarshalAggrAgreementMessage(r *bytes.Buffer, m SerializableMessage) error {
	aggro := newAggrAgreement()
	if err := UnmarshalAggrAgreement(r, aggro); err != nil {
		return err
	}

	m.SetPayload(*aggro)
	return nil
}
