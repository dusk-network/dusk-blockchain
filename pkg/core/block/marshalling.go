package block

import (
	"encoding/binary"
	"io"

	"github.com/dusk-network/dusk-blockchain/pkg/core/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
)

func Marshal(r io.Writer, b *Block) error {
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

func Unmarshal(r io.Reader, b *Block) error {
	if err := UnmarshalHeader(r, b.Header); err != nil {
		return err
	}

	lTxs, err := encoding.ReadVarInt(r)
	if err != nil {
		return err
	}

	b.Txs = make([]transactions.Transaction, lTxs)
	for i := range b.Txs {
		tx, err := transactions.Unmarshal(r)
		if err != nil {
			return err
		}
		b.Txs[i] = tx
	}

	return nil
}

func MarshalHashable(r io.Writer, h *Header) error {
	if err := encoding.WriteUint8(r, h.Version); err != nil {
		return err
	}

	if err := encoding.WriteUint64(r, binary.LittleEndian, h.Height); err != nil {
		return err
	}

	if err := encoding.WriteUint64(r, binary.LittleEndian, uint64(h.Timestamp)); err != nil {
		return err
	}

	if err := encoding.Write256(r, h.PrevBlockHash); err != nil {
		return err
	}

	if err := encoding.WriteBLS(r, h.Seed); err != nil {
		return err
	}

	return nil
}

func MarshalHeader(r io.Writer, h *Header) error {
	if err := MarshalHashable(r, h); err != nil {
		return err
	}

	if err := encoding.Write256(r, h.TxRoot); err != nil {
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

func UnmarshalHeader(r io.Reader, h *Header) error {
	if err := encoding.ReadUint8(r, &h.Version); err != nil {
		return err
	}

	if err := encoding.ReadUint64(r, binary.LittleEndian, &h.Height); err != nil {
		return err
	}

	var timestamp uint64
	if err := encoding.ReadUint64(r, binary.LittleEndian, &timestamp); err != nil {
		return err
	}
	h.Timestamp = int64(timestamp)

	if err := encoding.Read256(r, &h.PrevBlockHash); err != nil {
		return err
	}

	if err := encoding.ReadBLS(r, &h.Seed); err != nil {
		return err
	}

	if err := encoding.Read256(r, &h.TxRoot); err != nil {
		return err
	}

	if err := UnmarshalCertificate(r, h.Certificate); err != nil {
		return err
	}

	if err := encoding.Read256(r, &h.Hash); err != nil {
		return err
	}

	return nil
}

func MarshalCertificate(r io.Writer, c *Certificate) error {
	if err := encoding.WriteBLS(r, c.StepOneBatchedSig); err != nil {
		return err
	}

	if err := encoding.WriteBLS(r, c.StepTwoBatchedSig); err != nil {
		return err
	}

	if err := encoding.WriteUint8(r, c.Step); err != nil {
		return err
	}

	if err := encoding.WriteUint64(r, binary.LittleEndian, c.StepOneCommittee); err != nil {
		return err
	}

	if err := encoding.WriteUint64(r, binary.LittleEndian, c.StepTwoCommittee); err != nil {
		return err
	}

	return nil
}

func UnmarshalCertificate(r io.Reader, c *Certificate) error {
	if err := encoding.ReadBLS(r, &c.StepOneBatchedSig); err != nil {
		return err
	}

	if err := encoding.ReadBLS(r, &c.StepTwoBatchedSig); err != nil {
		return err
	}

	if err := encoding.ReadUint8(r, &c.Step); err != nil {
		return err
	}

	if err := encoding.ReadUint64(r, binary.LittleEndian, &c.StepOneCommittee); err != nil {
		return err
	}

	if err := encoding.ReadUint64(r, binary.LittleEndian, &c.StepTwoCommittee); err != nil {
		return err
	}

	return nil
}
