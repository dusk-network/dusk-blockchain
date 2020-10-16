package config

import (
	"bytes"
	"encoding/hex"
	"log"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/wallet"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/util/legacy"
)

// A single point of constants definition
const (
	// GeneratorReward is the amount of Block generator default reward
	// TODO: TBD
	GeneratorReward = 50 * wallet.DUSK

	MinFee = uint64(100)

	// MaxLockTime is the maximum amount of time a consensus transaction (stake, bid)
	// can be locked up for
	MaxLockTime = uint64(250000)

	// GenesisBlockBlob represents the genesis block bytes in hexadecimal format
	// It's recommended to be regenerated with generation.GenerateGensisBlock() API
	TestNetGenesisBlob = "0000000000000000002d9f6c5f000000000000000000000000000000000000000000000000000000000000000000000000900201a029d48838d25033ac36d7481f460e7ee340856a4fe3fcd9283e3207f476d6690df4b3f59371d63e0d402e122fe515b1702b9566304470433cc24c43ec990000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000b5023751139f6df9dc6e64c3c68b92226ca7e7e8a27c5846d60a18f262480670010000000001000000200000000000000000000000000000000000000000000000000000000000000000012000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000639818579a1f9bc0d328dd363df9719952003b4d71ca21b30f9f4a1caff838152000000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000050c300000000000064000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002abbc288a92acd1a67de417af3338c581912d9a41925e288eaf6ec517e3a3cc41c8833a00f2052a01000000"

	// Maximum number of blocks to be requested/delivered on a single syncing session with a peer
	MaxInvBlocks = 500
)

// DecodeGenesis marshals a genesis block into a buffer
func DecodeGenesis() *block.Block {
	if Get().Genesis.Legacy {
		g := legacy.DecodeGenesis()
		c, err := legacy.OldBlockToNewBlock(g)
		if err != nil {
			log.Panic(err)
		}

		return c
	}

	b := block.NewBlock()
	switch Get().General.Network {
	case "testnet": //nolint
		blob, err := hex.DecodeString(TestNetGenesisBlob)
		if err != nil {
			log.Panic(err)
		}

		var buf bytes.Buffer
		_, _ = buf.Write(blob)
		if uerr := message.UnmarshalBlock(&buf, b); uerr != nil {
			log.Panic(uerr)
		}

		sanityCheck(TestNetGenesisBlob, b)
	}
	return b
}

// nolint
func sanityCheck(genesis string, b *block.Block) {
	// sanity check the genesis block
	r := new(bytes.Buffer)
	if err := message.MarshalBlock(r, b); err != nil {
		log.Panic(err)
	}
	hgen := hex.EncodeToString(r.Bytes())
	if hgen != TestNetGenesisBlob {
		log.Panic("genesis blob is wrongly serialized")
	}
}
