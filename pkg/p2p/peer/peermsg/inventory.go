package peermsg

import (
	"fmt"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

type InvType uint8

var (
	InvTypeMempoolTx InvType = 0
	InvTypeBlock     InvType = 1

	supportedInvTypes = [2]InvType{
		InvTypeMempoolTx,
		InvTypeBlock,
	}
)

type InvVect struct {
	Type InvType // Type of data
	Hash []byte  // Hash of the data
}

type Inv struct {
	InvList []InvVect
}

func (i *Inv) Encode(w io.Writer) error {
	if err := encoding.WriteVarInt(w, uint64(len(i.InvList))); err != nil {
		return err
	}

	for _, vect := range i.InvList {
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

func (inv *Inv) Decode(r io.Reader) error {
	lenVect, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
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

		if err := encoding.Read256(r, &inv.InvList[i].Hash); err != nil {
			return err
		}
	}

	return nil
}

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
