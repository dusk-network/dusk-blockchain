// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package message_test

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/config/genesis"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	assert "github.com/stretchr/testify/require"
)

func TestEncodeDecodeBlock(t *testing.T) {
	assert := assert.New(t)

	// random block
	blk := helper.RandomBlock(200, 2)

	// Encode block into a buffer
	buf := new(bytes.Buffer)
	assert.NoError(message.MarshalBlock(buf, blk))

	// Decode buffer into a block struct
	decBlk := block.NewBlock()
	assert.NoError(message.UnmarshalBlock(buf, decBlk))

	// Check both structs are equal
	assert.True(blk.Equals(decBlk))

	// Check that all txs are equal
	for i := range blk.Txs {
		hash1, err := blk.Txs[i].CalculateHash()
		assert.NoError(err)

		hash2, err := decBlk.Txs[i].CalculateHash()
		assert.NoError(err)

		assert.True(bytes.Equal(hash1, hash2))
		assert.True(transactions.Equal(blk.Txs[i], decBlk.Txs[i]))
	}
}

func TestEncodeDecodeCert(t *testing.T) {
	assert := assert.New(t)

	// random certificate
	cert := helper.RandomCertificate()

	// Encode certificate into a buffer
	buf := new(bytes.Buffer)

	err := message.MarshalCertificate(buf, cert)
	assert.Nil(err)

	// Decode buffer into a certificate struct
	decCert := &block.Certificate{}

	err = message.UnmarshalCertificate(buf, decCert)
	assert.Nil(err)

	// Check both structs are equal
	assert.True(cert.Equals(decCert))
}

func TestEncodeDecodeHeader(t *testing.T) {
	assert := assert.New(t)

	// Create a random header
	hdr := helper.RandomHeader(200)

	hash, err := hdr.CalculateHash()
	assert.Nil(err)

	hdr.Hash = hash

	// Encode header into a buffer
	buf := new(bytes.Buffer)

	err = message.MarshalHeader(buf, hdr)
	assert.Nil(err)

	// Decode buffer into a header struct
	decHdr := block.NewHeader()

	err = message.UnmarshalHeader(buf, decHdr)
	assert.Nil(err)

	// Check both structs are equal
	assert.True(hdr.Equals(decHdr))
}

func TestDecodeLegacyGenesis(t *testing.T) { //nolint
	genesis.Decode()
}
