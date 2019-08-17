package consensus_test

import (
	"bytes"
	"encoding/binary"
	"math"
	"math/rand"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/msg"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/lite"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/core/transactions"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/stretchr/testify/assert"
)

func TestInitiate(t *testing.T) {
	bus := wire.NewEventBus()
	keys, _ := user.NewRandKeys()
	_, db := lite.CreateDBConnection()
	initChan := make(chan *bytes.Buffer, 1)
	bus.Subscribe(msg.InitializationTopic, initChan)

	if err := consensus.GetStartingRound(bus, db, keys); err != nil {
		t.Fatal(err)
	}

	blk := helper.RandomBlock(t, 1, 2)
	stake := makeStake(&keys)
	blk.AddTx(stake)

	buf := new(bytes.Buffer)
	if err := blk.Encode(buf); err != nil {
		t.Fatal(err)
	}

	bus.Publish(string(topics.AcceptedBlock), buf)
	round := <-initChan
	assert.Equal(t, uint64(2), binary.LittleEndian.Uint64(round.Bytes()))
}

func makeStake(keys *user.Keys) *transactions.Stake {
	R, _ := crypto.RandEntropy(32)

	stake, _ := transactions.NewStake(0, math.MaxUint64, 100, R, *keys.EdPubKey, keys.BLSPubKey.Marshal())
	rangeProof, _ := crypto.RandEntropy(32)
	stake.RangeProof = rangeProof
	keyImage, _ := crypto.RandEntropy(32)
	pubkey, _ := crypto.RandEntropy(32)
	pseudoComm, _ := crypto.RandEntropy(32)
	signature, _ := crypto.RandEntropy(32)
	input, _ := transactions.NewInput(keyImage, pubkey, pseudoComm, signature)
	stake.Inputs = transactions.Inputs{input}

	outputAmount := rand.Int63n(100000)
	commitment := make([]byte, 32)
	binary.BigEndian.PutUint64(commitment[24:32], uint64(outputAmount))
	destKey, _ := crypto.RandEntropy(32)
	output, _ := transactions.NewOutput(commitment, destKey)
	encryptedAmount, _ := crypto.RandEntropy(32)
	encryptedMask, _ := crypto.RandEntropy(32)
	output.EncryptedAmount = encryptedAmount
	output.EncryptedMask = encryptedMask
	stake.Outputs = transactions.Outputs{output}

	return stake
}
