package responding_test

import (
	"bytes"

	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/peermsg"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/responding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	crypto "github.com/dusk-network/dusk-crypto/hash"
)

func TestRequestData(t *testing.T) {
	_, db := lite.CreateDBConnection()
	defer func() {
		_ = db.Close()
	}()

	responseChan := make(chan *bytes.Buffer, 100)
	dataRequestor := responding.NewDataRequestor(db, nil, responseChan)

	// Send topics.Inv
	hash, buf, err := createInvBuffer()
	if err != nil {
		t.Fatal(err)
	}

	if err := dataRequestor.RequestMissingItems(buf); err != nil {
		t.Fatal(err)
	}

	response := <-responseChan

	// Check topic
	topic, _ := topics.Extract(response)
	if topic != topics.GetData {
		t.Fatalf("unexpected topic %s, expected GetData", topic)
	}

	// Assert the output
	inv := &peermsg.Inv{}
	if err := inv.Decode(response); err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(inv.InvList[0].Hash, hash) {
		t.Error("hash from topics.GetData is not equal to the hash from topics.Inv")
	}

	if len(inv.InvList) > 1 {
		t.Error("expecting only one object")
	}
}

func createInvBuffer() ([]byte, *bytes.Buffer, error) {
	msg := &peermsg.Inv{}
	hash, _ := crypto.RandEntropy(32)
	msg.AddItem(peermsg.InvTypeBlock, hash)

	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		return nil, nil, err
	}

	return hash, buf, nil
}
