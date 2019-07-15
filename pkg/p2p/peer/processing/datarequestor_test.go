package processing_test

import (
	"bytes"

	"testing"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database/lite"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/peermsg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/processing"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

func TestRequestData(t *testing.T) {
	_, db := lite.CreateDBConnection()
	defer db.Close()

	responseChan := make(chan *bytes.Buffer, 100)
	dataRequestor := processing.NewDataRequestor(db, nil, responseChan)

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
	topic := extractTopic(response)
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
