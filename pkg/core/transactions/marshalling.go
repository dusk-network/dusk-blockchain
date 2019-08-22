package transactions

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
)

func Unmarshal(r io.Reader) (Transaction, error) {
	var txType uint8
	if err := encoding.ReadUint8(r, &txType); err != nil {
		return nil, err
	}

	// Unmarshal
	switch TxType(txType) {
	case StandardType:
		tx := &Standard{TxType: TxType(txType)}
		err := UnmarshalStandard(r, tx)
		return tx, err
	case TimelockType:
		tx := NewTimeLock(0, 0, 0, nil)
		err := UnmarshalTimelock(r, tx)
		return tx, err
	case BidType:
		tx := &Bid{TimeLock: *NewTimeLock(0, 0, 0, nil)}
		tx.TxType = BidType
		err := UnmarshalBid(r, tx)
		return tx, err
	case StakeType:
		tx := &Stake{TimeLock: *NewTimeLock(0, 0, 0, nil)}
		tx.TxType = StakeType
		err := UnmarshalStake(r, tx)
		return tx, err
	case CoinbaseType:
		tx := &Coinbase{TxType: TxType(txType)}
		err := UnmarshalCoinbase(r, tx)
		return tx, err
	default:
		return nil, fmt.Errorf("unknown transaction type: %d", txType)
	}
}

func Marshal(r io.Writer, tx Transaction) error {
	switch tx.Type() {
	case StandardType:
		return MarshalStandard(r, tx.(*Standard))
	case TimelockType:
		return MarshalTimelock(r, tx.(*TimeLock))
	case BidType:
		return MarshalBid(r, tx.(*Bid))
	case StakeType:
		return MarshalStake(r, tx.(*Stake))
	case CoinbaseType:
		return MarshalCoinbase(r, tx.(*Coinbase))
	default:
		return fmt.Errorf("unknown transaction type: %d", tx.Type())
	}
}

func MarshalStandard(w io.Writer, s *Standard) error {
	if err := encoding.WriteUint8(w, uint8(s.TxType)); err != nil {
		return err
	}

	if err := encoding.Write256(w, s.R); err != nil {
		return err
	}

	if err := encoding.WriteUint8(w, uint8(s.Version)); err != nil {
		return err
	}

	if err := encoding.WriteVarInt(w, uint64(len(s.Inputs))); err != nil {
		return err
	}

	for _, input := range s.Inputs {
		if err := MarshalInput(w, input); err != nil {
			return err
		}
	}
	if err := encoding.WriteVarInt(w, uint64(len(s.Outputs))); err != nil {
		return err
	}

	for _, output := range s.Outputs {
		if err := MarshalOutput(w, output); err != nil {
			return err
		}
	}

	if err := encoding.WriteUint64(w, binary.LittleEndian, s.Fee); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(w, s.RangeProof); err != nil {
		return err
	}

	return nil
}

func MarshalTimelock(r io.Writer, tx *TimeLock) error {
	if err := MarshalStandard(r, &tx.Standard); err != nil {
		return err
	}

	if err := encoding.WriteUint64(r, binary.LittleEndian, tx.Lock); err != nil {
		return err
	}

	return nil
}

func MarshalBid(r io.Writer, tx *Bid) error {
	if err := MarshalTimelock(r, &tx.TimeLock); err != nil {
		return err
	}

	if err := encoding.Write256(r, tx.M); err != nil {
		return err
	}

	return nil
}

func MarshalStake(r io.Writer, tx *Stake) error {
	if err := MarshalTimelock(r, &tx.TimeLock); err != nil {
		return err
	}

	if err := encoding.Write256(r, tx.PubKeyEd); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(r, tx.PubKeyBLS); err != nil {
		return err
	}

	return nil
}

func MarshalCoinbase(w io.Writer, c *Coinbase) error {
	if err := encoding.WriteUint8(w, uint8(c.TxType)); err != nil {
		return err
	}

	if err := encoding.Write256(w, c.R); err != nil {
		return err
	}

	if err := encoding.Write256(w, c.Score); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(w, c.Proof); err != nil {
		return err
	}

	if err := encoding.WriteVarInt(w, uint64(len(c.Rewards))); err != nil {
		return err
	}

	for _, output := range c.Rewards {
		if err := MarshalOutput(w, output); err != nil {
			return err
		}
	}

	return nil
}

func UnmarshalStandard(w io.Reader, s *Standard) error {
	if err := encoding.Read256(w, &s.R); err != nil {
		return err
	}

	var ver uint8
	if err := encoding.ReadUint8(w, &ver); err != nil {
		return err
	}
	s.Version = ver

	lInputs, err := encoding.ReadVarInt(w)
	if err != nil {
		return err
	}

	s.Inputs = make(Inputs, lInputs)
	for i := range s.Inputs {
		s.Inputs[i] = &Input{}
		if err := UnmarshalInput(w, s.Inputs[i]); err != nil {
			return err
		}
	}

	lOutputs, err := encoding.ReadVarInt(w)
	if err != nil {
		return err
	}

	s.Outputs = make(Outputs, lOutputs)
	for i := range s.Outputs {
		s.Outputs[i] = &Output{}
		if err := UnmarshalOutput(w, s.Outputs[i]); err != nil {
			return err
		}
	}

	if err := encoding.ReadUint64(w, binary.LittleEndian, &s.Fee); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(w, &s.RangeProof); err != nil {
		return err
	}

	return nil
}

func UnmarshalTimelock(r io.Reader, tx *TimeLock) error {
	if err := UnmarshalStandard(r, &tx.Standard); err != nil {
		return err
	}

	if err := encoding.ReadUint64(r, binary.LittleEndian, &tx.Lock); err != nil {
		return err
	}

	return nil
}

func UnmarshalBid(r io.Reader, tx *Bid) error {
	if err := UnmarshalTimelock(r, &tx.TimeLock); err != nil {
		return err
	}

	if err := encoding.Read256(r, &tx.M); err != nil {
		return err
	}

	return nil
}

func UnmarshalStake(r io.Reader, tx *Stake) error {
	if err := UnmarshalTimelock(r, &tx.TimeLock); err != nil {
		return err
	}

	if err := encoding.Read256(r, &tx.PubKeyEd); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &tx.PubKeyBLS); err != nil {
		return err
	}

	return nil
}

func UnmarshalCoinbase(w io.Reader, c *Coinbase) error {
	if err := encoding.Read256(w, &c.R); err != nil {
		return err
	}

	if err := encoding.Read256(w, &c.Score); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(w, &c.Proof); err != nil {
		return err
	}

	lRewards, err := encoding.ReadVarInt(w)
	if err != nil {
		return err
	}

	c.Rewards = make(Outputs, lRewards)
	for i := range c.Rewards {
		c.Rewards[i] = &Output{}
		if err := UnmarshalOutput(w, c.Rewards[i]); err != nil {
			return err
		}
	}

	return nil
}
