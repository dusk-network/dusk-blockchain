// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package config

import (
	"bytes"
	"encoding/hex"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/wallet"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/sirupsen/logrus"
)

// A single point of constants definition.
const (
	// GeneratorReward is the amount of Block generator default reward.
	// TODO: TBD.
	GeneratorReward = 50 * wallet.DUSK

	MinFee = uint64(100)

	// MaxLockTime is the maximum amount of time a consensus transaction (stake, bid)
	// can be locked up for.
	MaxLockTime = uint64(250000)

	// Maximum number of blocks to be requested/delivered on a single syncing session with a peer.
	MaxInvBlocks = 500

	// Protocol-based consensus step time.
	ConsensusTimeOut = 5 * time.Second
)

// DecodeGenesis marshals a genesis block into a buffer.
func DecodeGenesis() *block.Block {
	b := block.NewBlock()

	switch Get().General.Network {
	case "testnet": //nolint
		genesis := ReadGenesis()

		blob, err := hex.DecodeString(genesis)
		if err != nil {
			logrus.Panic(err)
		}

		var buf bytes.Buffer
		_, _ = buf.Write(blob)

		if uerr := message.UnmarshalBlock(&buf, b); uerr != nil {
			log.Panic(uerr)
		}

		sanityCheck(genesis, b)
	}

	return b
}

// ReadGenesis from the configured path.
func ReadGenesis() string {
	path := os.ExpandEnv(r.General.GenesisPath)

	//nolint:gosec
	file, err := os.Open(path)
	if err != nil {
		logrus.WithError(err).Panic("could not open genesis block file")
	}

	//nolint
	defer file.Close()

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		logrus.WithError(err).Panic("could not read genesis block")
	}

	return string(bytes)
}

// nolint
func sanityCheck(genesis string, b *block.Block) {
	// sanity check the genesis block
	r := new(bytes.Buffer)
	if err := message.MarshalBlock(r, b); err != nil {
		log.Panic(err)
	}

	hgen := hex.EncodeToString(r.Bytes())
	if hgen != genesis {
		log.Panic("genesis blob is wrongly serialized")
	}
}
