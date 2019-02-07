package payload_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"
)

func TestMsgTxCoinbaseEncodeDecode(t *testing.T) {
	byte32 := []byte{1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4}

	sig, _ := crypto.RandEntropy(2000)

	txPubKey, _ := crypto.RandEntropy(32)
	cb := transactions.NewCoinbase(sig, txPubKey, byte32)
	s := transactions.NewTX(transactions.CoinbaseType, cb)
	s.R = txPubKey

	out := transactions.NewOutput(200, byte32, sig)
	cb.AddReward(out)
	if err := s.SetHash(); err != nil {
		t.Fatal(err)
	}

	msg := payload.NewMsgTx(s)

	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		t.Fatal(err)
	}

	msg2 := &payload.MsgTx{}
	if err := msg2.Decode(buf); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, s, msg2.Tx)
}

func TestMsgTxBidEncodeDecode(t *testing.T) {
	byte32 := []byte{1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4}

	sig, _ := crypto.RandEntropy(2000)

	txPubKey, _ := crypto.RandEntropy(32)
	bid := transactions.NewBid(100, txPubKey, 100)
	s := transactions.NewTX(transactions.BidType, bid)
	in := transactions.NewInput(txPubKey, txPubKey, 0, sig)
	bid.AddInput(in)
	s.R = txPubKey

	out := transactions.NewOutput(200, byte32, sig)
	bid.AddOutput(out)
	if err := s.SetHash(); err != nil {
		t.Fatal(err)
	}

	msg := payload.NewMsgTx(s)

	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		t.Fatal(err)
	}

	msg2 := &payload.MsgTx{}
	if err := msg2.Decode(buf); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, s, msg2.Tx)
}

func TestMsgTxStakeEncodeDecode(t *testing.T) {
	byte32 := []byte{1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4}

	sig, _ := crypto.RandEntropy(2000)

	txPubKey, _ := crypto.RandEntropy(32)
	stake := transactions.NewStake(100, 200, txPubKey, byte32)
	s := transactions.NewTX(transactions.StakeType, stake)
	in := transactions.NewInput(txPubKey, txPubKey, 0, sig)
	stake.AddInput(in)
	s.R = txPubKey

	out := transactions.NewOutput(200, byte32, sig)
	stake.AddOutput(out)
	if err := s.SetHash(); err != nil {
		t.Fatal(err)
	}

	msg := payload.NewMsgTx(s)

	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		t.Fatal(err)
	}

	msg2 := &payload.MsgTx{}
	if err := msg2.Decode(buf); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, s, msg2.Tx)
}

func TestMsgTxStandardEncodeDecode(t *testing.T) {
	byte32 := []byte{1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4}

	sig, _ := crypto.RandEntropy(2000)

	txPubKey, _ := crypto.RandEntropy(32)
	pl := transactions.NewStandard(100)
	s := transactions.NewTX(transactions.StandardType, pl)
	in := transactions.NewInput(txPubKey, txPubKey, 0, sig)
	pl.AddInput(in)
	s.R = txPubKey

	out := transactions.NewOutput(200, byte32, sig)
	pl.AddOutput(out)
	if err := s.SetHash(); err != nil {
		t.Fatal(err)
	}

	msg := payload.NewMsgTx(s)

	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		t.Fatal(err)
	}

	msg2 := &payload.MsgTx{}
	if err := msg2.Decode(buf); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, s, msg2.Tx)
}

func TestMsgTxTimelockEncodeDecode(t *testing.T) {
	byte32 := []byte{1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4}

	sig, _ := crypto.RandEntropy(2000)

	txPubKey, _ := crypto.RandEntropy(32)
	timelock := transactions.NewTimelock(100, 200)
	s := transactions.NewTX(transactions.TimelockType, timelock)
	in := transactions.NewInput(txPubKey, txPubKey, 0, sig)
	timelock.AddInput(in)
	s.R = txPubKey

	out := transactions.NewOutput(200, byte32, sig)
	timelock.AddOutput(out)
	if err := s.SetHash(); err != nil {
		t.Fatal(err)
	}

	msg := payload.NewMsgTx(s)

	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		t.Fatal(err)
	}

	msg2 := &payload.MsgTx{}
	if err := msg2.Decode(buf); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, s, msg2.Tx)
}

func TestMsgTxContractEncodeDecode(t *testing.T) {
	byte32 := []byte{1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4}

	sig, _ := crypto.RandEntropy(2000)

	txPubKey, _ := crypto.RandEntropy(32)
	contract := transactions.NewContract(100, 200)
	s := transactions.NewTX(transactions.ContractType, contract)
	in := transactions.NewInput(txPubKey, txPubKey, 0, sig)
	contract.AddInput(in)
	s.R = txPubKey

	out := transactions.NewOutput(200, byte32, sig)
	contract.AddOutput(out)
	if err := s.SetHash(); err != nil {
		t.Fatal(err)
	}

	msg := payload.NewMsgTx(s)

	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		t.Fatal(err)
	}

	msg2 := &payload.MsgTx{}
	if err := msg2.Decode(buf); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, s, msg2.Tx)
}
