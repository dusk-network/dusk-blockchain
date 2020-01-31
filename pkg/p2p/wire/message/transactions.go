package message

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"math/big"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-crypto/mlsag"
	"github.com/dusk-network/dusk-wallet/v2/transactions"
)

func UnmarshalTxMessage(r *bytes.Buffer, m SerializableMessage) error {
	tx, err := UnmarshalTx(r)
	if err != nil {
		return err
	}
	m.SetPayload(tx)
	return nil
}

func UnmarshalTx(r *bytes.Buffer) (transactions.Transaction, error) {
	var txType uint8
	if err := encoding.ReadUint8(r, &txType); err != nil {
		return nil, err
	}

	// Unmarshal
	switch transactions.TxType(txType) {
	case transactions.StandardType:
		tx := &transactions.Standard{TxType: transactions.TxType(txType)}
		err := UnmarshalStandard(r, tx)
		return tx, err
	case transactions.TimelockType:
		tx, err := transactions.NewTimelock(0, 0, 0, 0)
		if err != nil {
			return nil, err
		}

		err = UnmarshalTimelock(r, tx)
		return tx, err
	case transactions.BidType:
		tx, err := transactions.NewBid(0, 0, 0, 0, nil)
		if err != nil {
			return nil, err
		}

		err = UnmarshalBid(r, tx)
		return tx, err
	case transactions.StakeType:
		tx, err := transactions.NewStake(0, 0, 0, 0, nil, nil)
		if err != nil {
			return nil, err
		}

		err = UnmarshalStake(r, tx)
		return tx, err
	case transactions.CoinbaseType:
		tx := &transactions.Coinbase{TxType: transactions.TxType(txType)}
		err := UnmarshalCoinbase(r, tx)
		return tx, err
	default:
		return nil, fmt.Errorf("unknown transaction type: %d", txType)
	}
}

func MarshalTx(r *bytes.Buffer, tx transactions.Transaction) error {
	switch tx.Type() {
	case transactions.StandardType:
		return MarshalStandard(r, tx.(*transactions.Standard))
	case transactions.TimelockType:
		return MarshalTimelock(r, tx.(*transactions.Timelock))
	case transactions.BidType:
		return MarshalBid(r, tx.(*transactions.Bid))
	case transactions.StakeType:
		return MarshalStake(r, tx.(*transactions.Stake))
	case transactions.CoinbaseType:
		return MarshalCoinbase(r, tx.(*transactions.Coinbase))
	default:
		return fmt.Errorf("unknown transaction type: %d", tx.Type())
	}
}

func MarshalStandard(w *bytes.Buffer, s *transactions.Standard) error {
	return marshalStandard(w, s, true)
}

func marshalStandard(w *bytes.Buffer, s *transactions.Standard, encodeSignature bool) error {
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
	buf := &bytes.Buffer{}
	if err := s.RangeProof.Encode(buf, true); err != nil {
		return err
	}

	return encoding.WriteVarBytes(w, buf.Bytes())
}

func MarshalTimelock(r *bytes.Buffer, tx *transactions.Timelock) error {
	return marshalTimelock(r, tx, true)
}

func marshalTimelock(r *bytes.Buffer, tx *transactions.Timelock, encodeSignature bool) error {
	if err := marshalStandard(r, tx.Standard, encodeSignature); err != nil {
		return err
	}

	if err := encoding.WriteUint64LE(r, tx.Lock); err != nil {
		return err
	}

	return nil
}

func MarshalBid(r *bytes.Buffer, tx *transactions.Bid) error {
	return marshalBid(r, tx, true)
}

func marshalBid(r *bytes.Buffer, tx *transactions.Bid, encodeSignature bool) error {
	if err := marshalTimelock(r, tx.Timelock, encodeSignature); err != nil {
		return err
	}

	if err := encoding.Write256(r, tx.M); err != nil {
		return err
	}

	return nil
}

func MarshalStake(r *bytes.Buffer, tx *transactions.Stake) error {
	return marshalStake(r, tx, true)
}

func marshalStake(r *bytes.Buffer, tx *transactions.Stake, encodeSignature bool) error {
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

func MarshalCoinbase(w *bytes.Buffer, c *transactions.Coinbase) error {
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

func UnmarshalStandard(w *bytes.Buffer, s *transactions.Standard) error {
	RBytes := make([]byte, 32)
	if err := encoding.Read256(w, RBytes); err != nil {
		return err
	}
	s.R.UnmarshalBinary(RBytes)

	if err := encoding.ReadUint8(w, &s.Version); err != nil {
		return err
	}

	lInputs, err := encoding.ReadVarInt(w)
	if err != nil {
		return err
	}

	// Maximum amount of inputs we can decode at once is
	// math.MaxInt32 / 8, since they are pointers (uint64)
	if lInputs > (math.MaxInt32 / 8) {
		return errors.New("input count too large")
	}

	s.Inputs = make(transactions.Inputs, lInputs)
	for i := range s.Inputs {
		s.Inputs[i] = &transactions.Input{Proof: mlsag.NewDualKey(), Signature: &mlsag.Signature{}}
		if err := UnmarshalInput(w, s.Inputs[i]); err != nil {
			return err
		}
	}

	lOutputs, err := encoding.ReadVarInt(w)
	if err != nil {
		return err
	}

	// Same for outputs
	if lOutputs > (math.MaxInt32 / 8) {
		return errors.New("output count too large")
	}

	s.Outputs = make(transactions.Outputs, lOutputs)
	for i := range s.Outputs {
		s.Outputs[i] = &transactions.Output{}
		if err := UnmarshalOutput(w, s.Outputs[i]); err != nil {
			return err
		}
	}

	var fee uint64
	if err := encoding.ReadUint64LE(w, &fee); err != nil {
		return err
	}
	s.Fee.SetBigInt(big.NewInt(0).SetUint64(fee))

	var rangeProofBuf []byte
	if err := encoding.ReadVarBytes(w, &rangeProofBuf); err != nil {
		return err
	}
	return s.RangeProof.Decode(bytes.NewBuffer(rangeProofBuf), true)
}

func UnmarshalTimelock(r *bytes.Buffer, tx *transactions.Timelock) error {
	err := UnmarshalStandard(r, tx.Standard)
	if err != nil {
		return err
	}

	if err = encoding.ReadUint64LE(r, &tx.Lock); err != nil {
		return err
	}

	return nil
}

func UnmarshalBid(r *bytes.Buffer, tx *transactions.Bid) error {
	err := UnmarshalTimelock(r, tx.Timelock)
	if err != nil {
		return err
	}

	tx.M = make([]byte, 32)
	if err := encoding.Read256(r, tx.M); err != nil {
		return err
	}

	return nil
}

func UnmarshalStake(r *bytes.Buffer, tx *transactions.Stake) error {
	err := UnmarshalTimelock(r, tx.Timelock)
	if err != nil {
		return err
	}

	tx.PubKeyEd = make([]byte, 32)
	if err := encoding.Read256(r, tx.PubKeyEd); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(r, &tx.PubKeyBLS); err != nil {
		return err
	}

	return nil
}

func UnmarshalCoinbase(w *bytes.Buffer, c *transactions.Coinbase) error {
	RBytes := make([]byte, 32)
	if err := encoding.Read256(w, RBytes); err != nil {
		return err
	}
	c.R.UnmarshalBinary(RBytes)

	c.Score = make([]byte, 32)
	if err := encoding.Read256(w, c.Score); err != nil {
		return err
	}

	if err := encoding.ReadVarBytes(w, &c.Proof); err != nil {
		return err
	}

	lRewards, err := encoding.ReadVarInt(w)
	if err != nil {
		return err
	}

	// Maximum amount of rewards is math.MaxInt32 / 8, as they are
	// pointers to outputs.
	if lRewards > (math.MaxInt32 / 8) {
		return errors.New("too many rewards in coinbase tx")
	}

	c.Rewards = make(transactions.Outputs, lRewards)
	for i := range c.Rewards {
		c.Rewards[i] = &transactions.Output{}
		if err := UnmarshalOutput(w, c.Rewards[i]); err != nil {
			return err
		}
	}

	return nil
}

// Encode an Output struct and write to w.
func MarshalOutput(w *bytes.Buffer, o *transactions.Output) error {
	if err := encoding.Write256(w, o.Commitment.Bytes()); err != nil {
		return err
	}

	if err := encoding.Write256(w, o.PubKey.P.Bytes()); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(w, o.EncryptedAmount.Bytes()); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(w, o.EncryptedMask.Bytes()); err != nil {
		return err
	}

	return nil
}

// Decode an Output object from r into an output struct.
func UnmarshalOutput(r *bytes.Buffer, o *transactions.Output) error {
	commBytes := make([]byte, 32)
	if err := encoding.Read256(r, commBytes); err != nil {
		return err
	}
	o.Commitment.UnmarshalBinary(commBytes)

	pubKeyBytes := make([]byte, 32)
	if err := encoding.Read256(r, pubKeyBytes); err != nil {
		return err
	}
	o.PubKey.P.UnmarshalBinary(pubKeyBytes)

	var encAmountBytes []byte
	if err := encoding.ReadVarBytes(r, &encAmountBytes); err != nil {
		return err
	}
	o.EncryptedAmount.UnmarshalBinary(encAmountBytes)

	var encMaskBytes []byte
	if err := encoding.ReadVarBytes(r, &encMaskBytes); err != nil {
		return err
	}
	o.EncryptedMask.UnmarshalBinary(encMaskBytes)
	return nil
}

// MarshalInput marshals an Input object into a bytes.Buffer.
func MarshalInput(w *bytes.Buffer, i *transactions.Input, encodeSignature bool) error {
	if err := encoding.Write256(w, i.KeyImage.Bytes()); err != nil {
		return err
	}

	if err := encoding.Write256(w, i.PubKey.P.Bytes()); err != nil {
		return err
	}

	if err := encoding.Write256(w, i.PseudoCommitment.Bytes()); err != nil {
		return err
	}

	if encodeSignature {
		// Signature needs to be encoded and decoded as VarBytes, to ensure backwards-compatibility with
		// the current testnet.
		buf := new(bytes.Buffer)
		if err := i.Signature.Encode(buf, true); err != nil {
			return err
		}

		return encoding.WriteVarBytes(w, buf.Bytes())
	}
	return nil
}

// Decode an Input object from a bytes.Buffer.
func UnmarshalInput(r *bytes.Buffer, i *transactions.Input) error {
	keyImageBytes := make([]byte, 32)
	if err := encoding.Read256(r, keyImageBytes); err != nil {
		return err
	}
	i.KeyImage.UnmarshalBinary(keyImageBytes)

	pubKeyBytes := make([]byte, 32)
	if err := encoding.Read256(r, pubKeyBytes); err != nil {
		return err
	}
	i.PubKey.P.UnmarshalBinary(pubKeyBytes)

	pseudoCommBytes := make([]byte, 32)
	if err := encoding.Read256(r, pseudoCommBytes); err != nil {
		return err
	}
	i.PseudoCommitment.UnmarshalBinary(pseudoCommBytes)

	var sigBytes []byte
	if err := encoding.ReadVarBytes(r, &sigBytes); err != nil {
		return err
	}

	return i.Signature.Decode(bytes.NewBuffer(sigBytes), true)
}
