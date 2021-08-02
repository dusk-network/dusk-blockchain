// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package message

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message/payload"
	"github.com/sirupsen/logrus"
)

// InvType is a byte describing the Inventory type.
type InvType uint8

var (
	// InvTypeMempoolTx is the inventory type for unconfirmed Txs.
	InvTypeMempoolTx InvType = 0
	// InvTypeBlock is the inventory type for confirmed Txs.
	InvTypeBlock InvType = 1

	supportedInvTypes = [2]InvType{
		InvTypeMempoolTx,
		InvTypeBlock,
	}
)

// InvVect represents a request of sort for Inventory data.
type InvVect struct {
	Type InvType // Type of data
	Hash []byte  // Hash of the data
}

// Inv contains a list of Inventory vector.
type Inv struct {
	InvList []InvVect
}

// Copy an InvVect.
// Implements the payload.Safe interface.
func (i InvVect) Copy() payload.Safe {
	hash := make([]byte, len(i.Hash))
	copy(hash, i.Hash)

	return &InvVect{
		Type: i.Type,
		Hash: hash,
	}
}

// Copy an Inv.
// Implements the payload.Safe interface.
func (inv Inv) Copy() payload.Safe {
	list := make([]InvVect, len(inv.InvList))
	for i, item := range inv.InvList {
		list[i] = item.Copy().(InvVect)
	}

	return &Inv{list}
}

// Encode an Inventory request into a buffer.
func (inv *Inv) Encode(w *bytes.Buffer) error {
	// NOTE: this is hardcoded due to recursive imports. this needs to be fixed
	if uint32(len(inv.InvList)) > 10000 {
		return errors.New("inv message is too large")
	}

	if uint32(len(inv.InvList)) > 10 {
		logrus.WithField("list_size", len(inv.InvList)).Trace("encode inv message")
	}

	if err := encoding.WriteVarInt(w, uint64(len(inv.InvList))); err != nil {
		return err
	}

	for _, vect := range inv.InvList {
		if !supportedInvType(vect.Type) {
			return fmt.Errorf("not supported inventory data type %d", vect.Type)
		}

		if err := encoding.WriteUint8(w, uint8(vect.Type)); err != nil {
			return err
		}

		if len(vect.Hash) != 32 {
			return fmt.Errorf("invalid inventory data size %d", len(vect.Hash))
		}

		if err := encoding.Write256(w, vect.Hash); err != nil {
			return err
		}
	}

	return nil
}

// UnmarshalInvMessage into a SerializableMessage.
func UnmarshalInvMessage(r *bytes.Buffer, m SerializableMessage) error {
	inv := &Inv{}
	if err := inv.Decode(r); err != nil {
		return err
	}

	m.SetPayload(*inv)
	return nil
}

// Decode an Inventory from a buffer.
func (inv *Inv) Decode(r *bytes.Buffer) error {
	lenVect, e := encoding.ReadVarInt(r)
	if e != nil {
		return e
	}

	if lenVect > uint64(10000) {
		return errors.New("inv message is too large")
	}

	inv.InvList = make([]InvVect, lenVect)

	if lenVect > 10 {
		logrus.WithField("list_size", lenVect).Trace("decode inv message")
	}

	for i := uint64(0); i < lenVect; i++ {
		var invType uint8
		if err := encoding.ReadUint8(r, &invType); err != nil {
			return err
		}

		inv.InvList[i].Type = InvType(invType)

		if !supportedInvType(inv.InvList[i].Type) {
			return fmt.Errorf("not supported inventory data type %d", inv.InvList[i].Type)
		}

		inv.InvList[i].Hash = make([]byte, 32)
		if err := encoding.Read256(r, inv.InvList[i].Hash); err != nil {
			return err
		}
	}

	return nil
}

// AddItem to an Inventory.
func (inv *Inv) AddItem(t InvType, hash []byte) {
	item := InvVect{
		Type: t,
		Hash: hash,
	}

	inv.InvList = append(inv.InvList, item)
}

func supportedInvType(t InvType) bool {
	for _, s := range supportedInvTypes {
		if t == s {
			return true
		}
	}

	return false
}
