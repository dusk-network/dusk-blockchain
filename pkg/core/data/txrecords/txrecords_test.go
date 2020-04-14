package txrecords_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/txrecords"
	"gotest.tools/assert"
)

// Ensure integrity of data between encoding and decoding
func TestEncodeDecodeTxRecord(t *testing.T) {
	r := &txrecords.TxRecord{
		Direction:    txrecords.In,
		Timestamp:    time.Now().Unix(),
		Height:       500,
		TxType:       transactions.BidType,
		Amount:       7172727182793,
		UnlockHeight: 300000,
		Recipient:    "pippo",
	}

	buf := new(bytes.Buffer)
	if err := txrecords.Encode(buf, r); err != nil {
		t.Fatal(err)
	}

	decoded := &txrecords.TxRecord{}
	if err := txrecords.Decode(buf, decoded); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, r.Direction, decoded.Direction)
	assert.Equal(t, r.Timestamp, decoded.Timestamp)
	assert.Equal(t, r.Height, decoded.Height)
	assert.Equal(t, r.TxType, decoded.TxType)
	assert.Equal(t, r.Amount, decoded.Amount)
	assert.Equal(t, r.UnlockHeight, decoded.UnlockHeight)
	assert.Equal(t, r.Recipient, decoded.Recipient)
}
