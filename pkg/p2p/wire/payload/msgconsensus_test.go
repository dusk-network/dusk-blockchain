package payload

import (
	"testing"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
)

func TestMsgConsensusEncodeDecode(t *testing.T) {

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

	if _, err := NewMsgConsensus(10000, 29000, wrongByte32, byte32, byte32, nil); err == nil {
		t.Fatal("check for prevblockhash did not work")
	}

	if _, err := NewMsgConsensus(10000, 29000, byte32, wrongByte32, byte32, nil); err == nil {
		t.Fatal("check for sig did not work")
	}

	if _, err := NewMsgConsensus(10000, 29000, byte32, byte32, wrongByte32, nil); err == nil {
		t.Fatal("check for pk did not work")
	}
}
