// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package responding_test

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/responding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	crypto "github.com/dusk-network/dusk-crypto/hash"
)

func TestRequestData(t *testing.T) {
	_, db := lite.CreateDBConnection()

	defer func() {
		_ = db.Close()
	}()

	dataRequestor := responding.NewDataRequestor(db, nil)

	// Send topics.Inv
	hash, msg := createInv()

	bufs, err := dataRequestor.RequestMissingItems("", msg)
	if err != nil {
		t.Fatal(err)
	}

	// Check topic
	topic, _ := topics.Extract(&bufs[0])
	if topic != topics.GetData {
		t.Fatalf("unexpected topic %s, expected GetData", topic)
	}

	// Assert the output
	inv := &message.Inv{}
	if err := inv.Decode(&bufs[0]); err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(inv.InvList[0].Hash, hash) {
		t.Error("hash from topics.GetData is not equal to the hash from topics.Inv")
	}

	if len(inv.InvList) > 1 {
		t.Error("expecting only one object")
	}
}

func createInv() ([]byte, message.Message) {
	msg := &message.Inv{}
	hash, _ := crypto.RandEntropy(32)
	msg.AddItem(message.InvTypeBlock, hash)
	return hash, message.New(topics.Inv, *msg)
}
