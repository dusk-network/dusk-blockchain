package payload

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/transactions"
)

func TestMsgTxEncodeDecode(t *testing.T) {
	byte32 := []byte{1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4}

	// Input
	sig, _ := crypto.RandEntropy(2000)
	in := transactions.Input{
		KeyImage:  byte32,
		TxID:      byte32,
		Index:     1,
		Signature: sig,
	}

	// Output
	out := transactions.Output{
		Amount: 200,
		P:      byte32,
	}

	// Type attribute
	ta := transactions.TypeAttributes{
		Inputs:   []transactions.Input{in},
		TxPubKey: byte32,
		Outputs:  []transactions.Output{out},
	}

	R, _ := crypto.RandEntropy(32)
	s := transactions.Stealth{
		Version: 1,
		Type:    1,
		R:       R,
		TA:      ta,
	}

	msg := NewMsgTx(s)

	size := s.GetEncodeSize()
	bs := make([]byte, 0, size)
	buf := bytes.NewBuffer(bs)

	if err := msg.Encode(buf); err != nil {
		t.Fatal(err)
	}

	msg2 := MsgTx{}
	if err := msg2.Decode(buf); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, s, msg2.Tx)
}
