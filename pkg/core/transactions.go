package core

import (
	"bytes"
	"errors"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"
)

func (b *Blockchain) verifyCoinbase(c *transactions.Coinbase) error {
	return nil
}

func (b *Blockchain) verifyBid(bid *transactions.Bid) error {
	if bid.Timelock-b.height > maxLockTime {
		return errors.New("locktime too high")
	}

	// Check inputs and outputs
	outputs := make([]*transactions.Output, 1)
	outputs = append(outputs, bid.Output)
	if err := b.checkInputsOutputs(bid.Inputs, outputs, bid.Fee); err != nil {
		return err
	}

	return nil
}

func (b *Blockchain) verifyStake(s *transactions.Stake) error {
	if s.Timelock-b.height > maxLockTime {
		return errors.New("locktime too high")
	}

	// Check inputs and outputs
	outputs := make([]*transactions.Output, 1)
	outputs = append(outputs, s.Output)
	if err := b.checkInputsOutputs(s.Inputs, outputs, s.Fee); err != nil {
		return err
	}

	// Check for deanonymized output???

	return nil
}

func (b *Blockchain) verifyStandard(s *transactions.Standard) error {
	// Check inputs and outputs
	if err := b.checkInputsOutputs(s.Inputs, s.Outputs, s.Fee); err != nil {
		return err
	}

	return nil
}

func (b *Blockchain) verifyTimelock(t *transactions.Timelock) error {
	if t.Timelock-b.height > maxLockTime {
		return errors.New("locktime too high")
	}

	// Check inputs and outputs
	if err := b.checkInputsOutputs(t.Inputs, t.Outputs, t.Fee); err != nil {
		return err
	}

	return nil
}

func (b *Blockchain) verifyContract(c *transactions.Contract) error {
	if c.Timelock-b.height > maxLockTime {
		return errors.New("locktime too high")
	}

	// Check inputs and outputs
	if err := b.checkInputsOutputs(c.Inputs, c.Outputs, c.Fee); err != nil {
		return err
	}

	return nil
}

func (b *Blockchain) checkInputsOutputs(inputs []*transactions.Input, outputs []*transactions.Output, fee uint64) error {
	// TODO: check input ownership as well
	// Get total input amount
	var inAmount uint64
	for _, input := range inputs {
		key := append(database.TX, input.TxID...)
		value, err := b.db.Get(key)
		if err != nil {
			return err
		}

		tx := &transactions.Stealth{}
		buf := bytes.NewBuffer(value)
		if err := tx.Decode(buf); err != nil {
			return err
		}

		t := tx.TypeInfo.Type()
		switch t {
		case transactions.CoinbaseType:
			c := tx.TypeInfo.(*transactions.Coinbase)
			inAmount += c.Rewards[input.Index].Amount
		case transactions.BidType:
			bid := tx.TypeInfo.(*transactions.Bid)
			inAmount += bid.Output.Amount
		case transactions.StakeType:
			s := tx.TypeInfo.(*transactions.Stake)
			inAmount += s.Output.Amount
		case transactions.StandardType:
			s := tx.TypeInfo.(*transactions.Standard)
			inAmount += s.Outputs[input.Index].Amount
		case transactions.TimelockType:
			t := tx.TypeInfo.(*transactions.Timelock)
			inAmount += t.Outputs[input.Index].Amount
		case transactions.ContractType:
			c := tx.TypeInfo.(*transactions.Contract)
			inAmount += c.Outputs[input.Index].Amount
		}
	}

	// Now get total output amount
	var outAmount uint64
	for _, output := range outputs {
		outAmount += output.Amount
	}

	if outAmount+fee > inAmount {
		return errors.New("output amount too high/input amount too low")
	}

	return nil
}
