package payload_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"
)

func TestMsgTxEncodeDecode(t *testing.T) {
	byte32 := []byte{1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4}

	sig, _ := crypto.RandEntropy(2000)

	txPubKey, _ := crypto.RandEntropy(32)
	s := transactions.NewTX(transactions.StandardType, nil)
	in := transactions.NewInput(txPubKey, txPubKey, 0, sig)
	s.AddInput(in)
	s.AddTxPubKey(txPubKey)

	out := transactions.NewOutput(200, byte32, sig)
	s.AddOutput(out)
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
	stake := transactions.NewStake(txPubKey, byte32)
	s := transactions.NewTX(transactions.StakeType, stake)
	in := transactions.NewInput(txPubKey, txPubKey, 0, sig)
	s.AddInput(in)
	s.AddTxPubKey(txPubKey)

	out := transactions.NewOutput(200, byte32, sig)
	s.AddOutput(out)
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
