package payload_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
)

func TestMsgConsensusEncodeDecode(t *testing.T) {
	// Make dummy data
	sigBLS, err := crypto.RandEntropy(33)
	if err != nil {
		t.Fatal(err)
	}

	byte32, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	sigEd, err := crypto.RandEntropy(64)
	if err != nil {
		t.Fatal(err)
	}

	sigs, err := crypto.RandEntropy(200)
	if err != nil {
		t.Fatal(err)
	}

	// Agreement
	pl, err := consensusmsg.NewAgreement(sigBLS, false, 1, byte32, sigBLS, byte32)
	if err != nil {
		t.Fatal(err)
	}

	msg, err := payload.NewMsgConsensus(10000, 29000, byte32, sigEd, byte32, pl)
	if err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		t.Fatal(err)
	}

	msg2 := &payload.MsgConsensus{}
	if err := msg2.Decode(buf); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, msg, msg2)

	// Reduction
	pl2, err := consensusmsg.NewReduction(sigBLS, 1, byte32, sigBLS, byte32)
	if err != nil {
		t.Fatal(err)
	}

	msg3, err := payload.NewMsgConsensus(10000, 29000, byte32, sigEd, byte32, pl2)
	if err != nil {
		t.Fatal(err)
	}

	buf2 := new(bytes.Buffer)
	if err := msg3.Encode(buf2); err != nil {
		t.Fatal(err)
	}

	msg4 := &payload.MsgConsensus{}
	if err := msg4.Decode(buf2); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, msg3, msg4)

	// Signature Set Candidate
	pl3, err := consensusmsg.NewSigSetCandidate(byte32, sigs, byte32, sigBLS)
	if err != nil {
		t.Fatal(err)
	}

	msg5, err := payload.NewMsgConsensus(10000, 29000, byte32, sigEd, byte32, pl3)
	if err != nil {
		t.Fatal(err)
	}

	buf3 := new(bytes.Buffer)
	if err := msg5.Encode(buf3); err != nil {
		t.Fatal(err)
	}

	msg6 := &payload.MsgConsensus{}
	if err := msg6.Decode(buf3); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, msg5, msg6)

	// Signature Set Vote
	pl4, err := consensusmsg.NewSigSetVote(1, byte32, byte32, sigBLS, byte32, sigBLS)
	if err != nil {
		t.Fatal(err)
	}

	msg7, err := payload.NewMsgConsensus(10000, 29000, byte32, sigEd, byte32, pl4)
	if err != nil {
		t.Fatal(err)
	}

	buf4 := new(bytes.Buffer)
	if err := msg7.Encode(buf4); err != nil {
		t.Fatal(err)
	}

	msg8 := &payload.MsgConsensus{}
	if err := msg8.Decode(buf4); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, msg7, msg8)

	// TODO: implement other payload types when possible
}

func TestMsgConsensusChecks(t *testing.T) {
	byte32, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	wrongByte32, err := crypto.RandEntropy(33)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := payload.NewMsgConsensus(10000, 29000, wrongByte32, byte32, byte32, nil); err == nil {
		t.Fatal("check for prevblockhash did not work")
	}

	if _, err := payload.NewMsgConsensus(10000, 29000, byte32, wrongByte32, byte32, nil); err == nil {
		t.Fatal("check for sig did not work")
	}

	if _, err := payload.NewMsgConsensus(10000, 29000, byte32, byte32, wrongByte32, nil); err == nil {
		t.Fatal("check for pk did not work")
	}
}
