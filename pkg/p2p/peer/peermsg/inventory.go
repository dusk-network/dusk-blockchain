package peermsg

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
)

// InvType is a byte describing the Inventory type
type InvType uint8

var (
	//InvTypeMempoolTx is the inventory type for unconfirmed Txs
	InvTypeMempoolTx InvType = 0
	// InvTypeBlock is the inventory type for confirmed Txs
	InvTypeBlock InvType = 1

	supportedInvTypes = [2]InvType{
		InvTypeMempoolTx,
		InvTypeBlock,
	}
)

// InvVect represents a request of sort for Inventory data
type InvVect struct {
	Type InvType // Type of data
	Hash []byte  // Hash of the data
}

// Inv contains a list of Inventory vector
type Inv struct {
	InvList []InvVect
}

// Encode an Inventory request into a buffer
func (inv *Inv) Encode(w *bytes.Buffer) error {
	if uint32(len(inv.InvList)) > config.Get().Mempool.MaxInvItems {
		return errors.New("inv message is too large")
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

// Decode an Inventory from a buffer
func (inv *Inv) Decode(r *bytes.Buffer) error {
	lenVect, e := encoding.ReadVarInt(r)
	if e != nil {
		return e
	}

	if lenVect > uint64(config.Get().Mempool.MaxInvItems) {
		return errors.New("inv message is too large")
	}

	inv.InvList = make([]InvVect, lenVect)
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

// AddItem to an Inventory
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
