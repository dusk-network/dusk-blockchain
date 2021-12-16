// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package candidate

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/config/genesis"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
)

func TestGenerateGenesis(t *testing.T) {
	rpcBus := rpcbus.New()

	provideMempoolTxs(rpcBus)

	seed := make([]byte, 64)
	if _, err := rand.Read(seed); err != nil {
		t.Fatal(err.Error())
	}

	// Generate a new genesis block with new wallet pubkey
	genesisHex, err := GenerateGenesisBlock(&consensus.Emitter{RPCBus: rpcBus})
	if err != nil {
		t.Fatalf("expecting valid genesis block: %s", err.Error())
	}

	// Decode the result hex value to ensure it's a valid block
	blob, err := hex.DecodeString(genesisHex)
	if err != nil {
		t.Fatalf("expecting valid hex %s", err.Error())
	}

	var buf bytes.Buffer
	buf.Write(blob)

	b := block.NewBlock()
	if err := message.UnmarshalBlock(&buf, b); err != nil {
		t.Fatalf("expecting decodable hex %s", err.Error())
	}
}

// Print blob
// t.Logf("genesis: %s", genesisHex)

func TestGenesisBlock(t *testing.T) {
	// read the hard-coded genesis blob for testnet
	b := genesis.Decode()

	// sanity checks
	if b.Header.Height != 0 {
		t.Fatalf("expecting valid height in cfg.TestNetGenesisBlob")
	}

	if b.Header.Version != 0 {
		t.Fatalf("expecting valid version in cfg.TestNetGenesisBlob")
	}

	if b.Txs[len(b.Txs)-1].Type() != transactions.Distribute {
		t.Fatalf("expecting coinbase tx in cfg.TestNetGenesisBlob")
	}
}

func provideMempoolTxs(rpcBus *rpcbus.RPCBus) {
	c := make(chan rpcbus.Request, 1)
	if err := rpcBus.Register(topics.GetMempoolTxsBySize, c); err != nil {
		panic(err)
	}

	go func() {
		r := <-c
		r.RespChan <- rpcbus.NewResponse([]transactions.ContractCall{}, nil)
	}()
}
