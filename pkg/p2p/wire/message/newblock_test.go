// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package message_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/dusk-network/bls12_381-sign/go/cgo/bls"
	"github.com/dusk-network/dusk-blockchain/pkg/config/genesis"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

func binary(val byte, size int) []byte {
	buf := make([]byte, size)
	for i := range buf {
		buf[i] = val
	}

	return buf
}

// Test both the Marshal and Unmarshal functions, making sure the data is correctly
// stored to/retrieved from a Buffer.
func TestUnMarshal(t *testing.T) {
	// random block
	blk := helper.RandomBlock(200, 0)

	hdr := header.Header{
		BlockHash: blk.Header.Hash,
		Round:     999999,
		Step:      255,
		PubKeyBLS: make([]byte, 96),
	}

	result := "hash=vec!["
	for i, v := range blk.Header.Hash {
		result += fmt.Sprintf("%d", v)

		if i != len(blk.Header.Hash)-1 {
			result += ","
		}
	}

	result += "];"

	t.Log(result)

	hdr.BlockHash = blk.Header.Hash

	// mock candidate
	// scr := message.MockNewBlock(hdr, *blk)

	hdr = header.Header{Round: 99999, Step: 123, BlockHash: hdr.BlockHash, PubKeyBLS: make([]byte, 96)}
	r := new(bytes.Buffer)
	_ = header.MarshalSignableVote(r, hdr)

	keys := key.NewRandKeys()

	_, err := bls.Sign(keys.BLSSecretKey, keys.BLSPubKey, r.Bytes())
	if err != nil {
		panic(err)
	}

	scr := message.NewAgreement(hdr)
	scr.SetSignature(binary(5, 48))
	scr.VotesPerStep = make([]*message.StepVotes, 2)
	scr.VotesPerStep[0] = &message.StepVotes{BitSet: 11111, Signature: binary(1, 48)}
	scr.VotesPerStep[1] = &message.StepVotes{BitSet: 22222, Signature: binary(2, 48)}

	msg := message.NewAggrAgreement(*scr, 100, binary(7, 48))

	m := message.New(topics.AggrAgreement, msg)

	buf, err := message.Marshal(m)
	if err != nil {
		panic(err)
	}

	serialized := message.New(m.Category(), buf)

	buf2 := serialized.Payload().(message.SafeBuffer)

	// Create the listener and contact the voucher seeder
	gossip := protocol.NewGossip()

	gossip.Process(&buf2.Buffer)

	t.Log(buf2.Len())

	result = "vec!["
	for i, v := range buf2.Bytes() {
		result += fmt.Sprintf("%d", v)

		if i != buf2.Len()-1 {
			result += ","
		}
	}

	result += "];"

	t.Log(result)
}

func TestDeepCopy(t *testing.T) {
	hash, _ := crypto.RandEntropy(32)
	hdr := header.Mock()
	hdr.BlockHash = hash

	// mock candidate
	genesis := genesis.Decode()
	se := message.MockNewBlock(hdr, *genesis)

	buf := new(bytes.Buffer)
	assert.NoError(t, message.MarshalNewBlock(buf, se))

	deepCopy := se.Copy().(message.NewBlock)
	buf2 := new(bytes.Buffer)
	assert.NoError(t, message.MarshalNewBlock(buf2, deepCopy))

	assert.Equal(t, buf.Bytes(), buf2.Bytes())
}
