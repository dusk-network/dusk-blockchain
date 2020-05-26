package config

import (
	"bytes"
	"encoding/hex"
	"log"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/wallet"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/util/legacy"
)

// A signle point of constants definition
const (
	// GeneratorReward is the amount of Block generator default reward
	// TODO: TBD
	GeneratorReward = 50 * wallet.DUSK

	// ConsensusTimeOut is the time out for consensus step timers.
	ConsensusTimeOut = 5 * time.Second

	MinFee = uint64(100)

	// MaxLockTime is the maximum amount of time a consensus transaction (stake, bid)
	// can be locked up for
	MaxLockTime = uint64(250000)

	// GenesisBlockBlob represents the genesis block bytes in hexadecimal format
	// It's recommended to be regenerated with generation.GenerateGensisBlock() API
	TestNetGenesisBlob = "0000000000000000007c44be5e00000000000000000000000000000000000000000000000000000000000000000000000098f27a1af528a8f953891b1cbaafd517f6be1b1b5db4235cf146b4a1f17e4e34eee5478f3cd2266eedbf00d6f663967297691d1be44729e11ccb44022e7e2e39080000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000e83c2a70070f1ae311f535c5fc33b2209bf8227a2536b32b737446a90a1a1263010100f2052a01000000010101002061c36e407ac91f20174572eec95f692f5cff1c40bacd1b9f86c7fa7202e93bb620753c2f424caf3c9220876e8cfe0afdff7ffd7c984d5c7d95fa0b46cf3781d883"
)

// DecodeGenesis marshals a genesis block into a buffer
func DecodeGenesis() *block.Block {
	if Get().Genesis.Legacy {
		return legacy.DecodeGenesis()
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
