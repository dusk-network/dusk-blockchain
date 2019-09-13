package block

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/wallet/transactions"
)

func Marshal(r *bytes.Buffer, b *Block) error {
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

func Unmarshal(r *bytes.Buffer, b *Block) error {
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

func MarshalHashable(r *bytes.Buffer, h *Header) error {
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

	if err := encoding.WriteBLS(r, h.Seed); err != nil {
		return err
	}

	return nil
}

func MarshalHeader(r *bytes.Buffer, h *Header) error {
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

func UnmarshalHeader(r *bytes.Buffer, h *Header) error {
	var err error
	h.Version, err = encoding.ReadUint8(r)
	if err != nil {
		return err
	}

	h.Height, err = encoding.ReadUint64LE(r)
	if err != nil {
		return err
	}

	timestamp, err := encoding.ReadUint64LE(r)
	if err != nil {
		return err
	}
	h.Timestamp = int64(timestamp)

	h.PrevBlockHash, err = encoding.Read256(r)
	if err != nil {
		return err
	}

	h.Seed, err = encoding.ReadBLS(r)
	if err != nil {
		return err
	}

	h.TxRoot, err = encoding.Read256(r)
	if err != nil {
		return err
	}

	if err := UnmarshalCertificate(r, h.Certificate); err != nil {
		return err
	}

	h.Hash, err = encoding.Read256(r)
	if err != nil {
		return err
	}

	return nil
}

func MarshalCertificate(r *bytes.Buffer, c *Certificate) error {
	if err := encoding.WriteBLS(r, c.StepOneBatchedSig); err != nil {
		return err
	}

	if err := encoding.WriteBLS(r, c.StepTwoBatchedSig); err != nil {
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

func UnmarshalCertificate(r *bytes.Buffer, c *Certificate) error {
	var err error
	c.StepOneBatchedSig, err = encoding.ReadBLS(r)
	if err != nil {
		return err
	}

	c.StepTwoBatchedSig, err = encoding.ReadBLS(r)
	if err != nil {
		return err
	}

	c.Step, err = encoding.ReadUint8(r)
	if err != nil {
		return err
	}

	c.StepOneCommittee, err = encoding.ReadUint64LE(r)
	if err != nil {
		return err
	}

	c.StepTwoCommittee, err = encoding.ReadUint64LE(r)
	if err != nil {
		return err
	}

	return nil
}
