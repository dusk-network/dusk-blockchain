package transactions

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-crypto/mlsag"
)

func Unmarshal(r *bytes.Buffer) (Transaction, error) {
	txType, err := encoding.ReadUint8(r)
	if err != nil {
		return nil, err
	}

	// Unmarshal
	switch TxType(txType) {
	case StandardType:
		tx := &Standard{TxType: TxType(txType)}
		err := UnmarshalStandard(r, tx)
		return tx, err
	case TimelockType:
		tx, err := NewTimelock(0, 0, 0, 0)
		if err != nil {
			return nil, err
		}

		err = UnmarshalTimelock(r, tx)
		return tx, err
	case BidType:
		tx, err := NewBid(0, 0, 0, 0, nil)
		if err != nil {
			return nil, err
		}

		err = UnmarshalBid(r, tx)
		return tx, err
	case StakeType:
		tx, err := NewStake(0, 0, 0, 0, nil, nil)
		if err != nil {
			return nil, err
		}

		err = UnmarshalStake(r, tx)
		return tx, err
	case CoinbaseType:
		tx := &Coinbase{TxType: TxType(txType)}
		err := UnmarshalCoinbase(r, tx)
		return tx, err
	default:
		return nil, fmt.Errorf("unknown transaction type: %d", txType)
	}
}

func Marshal(r *bytes.Buffer, tx Transaction) error {
	switch tx.Type() {
	case StandardType:
		return MarshalStandard(r, tx.(*Standard))
	case TimelockType:
		return MarshalTimelock(r, tx.(*Timelock))
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

func MarshalStandard(w *bytes.Buffer, s *Standard) error {
	return marshalStandard(w, s, true)
}

func marshalStandard(w *bytes.Buffer, s *Standard, encodeSignature bool) error {
	if err := encoding.WriteUint8(w, uint8(s.TxType)); err != nil {
		return err
	}

	if err := encoding.Write256(w, s.R.Bytes()); err != nil {
		return err
	}

	if err := encoding.WriteUint8(w, uint8(s.Version)); err != nil {
		return err
	}

	if err := encoding.WriteVarInt(w, uint64(len(s.Inputs))); err != nil {
		return err
	}

	for _, input := range s.Inputs {
		if err := MarshalInput(w, input, encodeSignature); err != nil {
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

	if err := encoding.WriteUint64LE(w, s.Fee.BigInt().Uint64()); err != nil {
		return err
	}

	// Marshal the rangeproof into it's own buffer and then marshal it as VarBytes.
	// This is because Rangeproof.Decode uses `buf.ReadFrom`, which can cause issues
	// when trying to unmarshal a block with multiple txs.
	// It also ensures backwards-compatibility with the current testnet.
	buf := new(bytes.Buffer)
	if err := s.RangeProof.Encode(buf, true); err != nil {
		return err
	}

	return encoding.WriteVarBytes(w, buf.Bytes())
}

func MarshalTimelock(r *bytes.Buffer, tx *Timelock) error {
	return marshalTimelock(r, tx, true)
}

func marshalTimelock(r *bytes.Buffer, tx *Timelock, encodeSignature bool) error {
	if err := marshalStandard(r, tx.Standard, encodeSignature); err != nil {
		return err
	}

	if err := encoding.WriteUint64LE(r, tx.Lock); err != nil {
		return err
	}

	return nil
}

func MarshalBid(r *bytes.Buffer, tx *Bid) error {
	return marshalBid(r, tx, true)
}

func marshalBid(r *bytes.Buffer, tx *Bid, encodeSignature bool) error {
	if err := marshalTimelock(r, tx.Timelock, encodeSignature); err != nil {
		return err
	}

	if err := encoding.Write256(r, tx.M); err != nil {
		return err
	}

	return nil
}

func MarshalStake(r *bytes.Buffer, tx *Stake) error {
	return marshalStake(r, tx, true)
}

func marshalStake(r *bytes.Buffer, tx *Stake, encodeSignature bool) error {
	if err := marshalTimelock(r, tx.Timelock, encodeSignature); err != nil {
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

func MarshalCoinbase(w *bytes.Buffer, c *Coinbase) error {
	if err := encoding.WriteUint8(w, uint8(c.TxType)); err != nil {
		return err
	}

	if err := encoding.Write256(w, c.R.Bytes()); err != nil {
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

func UnmarshalStandard(w *bytes.Buffer, s *Standard) error {
	RBytes, err := encoding.Read256(w)
	if err != nil {
		return err
	}
	s.R.UnmarshalBinary(RBytes)

	ver, err := encoding.ReadUint8(w)
	if err != nil {
		return err
	}
	s.Version = ver

	lInputs, err := encoding.ReadVarInt(w)
	if err != nil {
		return err
	}

	s.Inputs = make(Inputs, lInputs)
	for i := range s.Inputs {
		s.Inputs[i] = &Input{Proof: mlsag.NewDualKey(), Signature: &mlsag.Signature{}}
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

	fee, err := encoding.ReadUint64LE(w)
	if err != nil {
		return err
	}
	s.Fee.SetBigInt(big.NewInt(0).SetUint64(fee))

	rangeProofBuf, err := encoding.ReadVarBytes(w)
	if err != nil {
		return err
	}
	return s.RangeProof.Decode(bytes.NewBuffer(rangeProofBuf), true)
}

func UnmarshalTimelock(r *bytes.Buffer, tx *Timelock) error {
	err := UnmarshalStandard(r, tx.Standard)
	if err != nil {
		return err
	}

	tx.Lock, err = encoding.ReadUint64LE(r)
	if err != nil {
		return err
	}

	return nil
}

func UnmarshalBid(r *bytes.Buffer, tx *Bid) error {
	err := UnmarshalTimelock(r, tx.Timelock)
	if err != nil {
		return err
	}

	tx.M, err = encoding.Read256(r)
	if err != nil {
		return err
	}

	return nil
}

func UnmarshalStake(r *bytes.Buffer, tx *Stake) error {
	err := UnmarshalTimelock(r, tx.Timelock)
	if err != nil {
		return err
	}

	tx.PubKeyEd, err = encoding.Read256(r)
	if err != nil {
		return err
	}

	tx.PubKeyBLS, err = encoding.ReadVarBytes(r)
	if err != nil {
		return err
	}

	return nil
}

func UnmarshalCoinbase(w *bytes.Buffer, c *Coinbase) error {
	RBytes, err := encoding.Read256(w)
	if err != nil {
		return err
	}
	c.R.UnmarshalBinary(RBytes)

	c.Score, err = encoding.Read256(w)
	if err != nil {
		return err
	}

	c.Proof, err = encoding.ReadVarBytes(w)
	if err != nil {
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
