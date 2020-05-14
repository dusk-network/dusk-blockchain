package config

import (
	"bytes"
	"encoding/hex"
	"log"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/wallet"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
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
	TestNetGenesisBlob = "0000000000000000001310bd5e000000000000000000000000000000000000000000000000000000000000000000000000a6088f341e69e4b4d6eaa34fc9734d084c7845245e1e01627f146518ffdbe8c72173123aa0646305c5d056e27bad06d715fecd3cf3cef97a5eb6b26fa7f438ca6f0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c1ba292067d2147b7dcff3cdcc54b9ad16477a07560beac0f4b2786d08622db4010100010000000000000000000211220233440233440255661800000000000000000000000000000000000000000000000000f2052a01000000023344023344000000000000000002556600000000000000000002112202334402334402556618000000000000000000000000000000000000000000000000000000000000000002334402334400000000000000000255660000010101002061c36e407ac91f20174572eec95f692f5cff1c40bacd1b9f86c7fa7202e93bb620753c2f424caf3c9220876e8cfe0afdff7ffd7c984d5c7d95fa0b46cf3781d883"
)

// DecodeGenesis marshals a genesis block into a buffer
func DecodeGenesis() *block.Block {
	b := block.NewBlock()
	switch Get().General.Network {
	case "testnet": //nolint
		blob, err := hex.DecodeString(TestNetGenesisBlob)
		if err != nil {
			log.Panic(err)
		}

		var buf bytes.Buffer
		_, _ = buf.Write(blob)
		if err := message.UnmarshalBlock(&buf, b); err != nil {
			log.Panic(err)
		}
	}
	return b
}
