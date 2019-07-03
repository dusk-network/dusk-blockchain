package consensus_test

import (
	"bytes"
	"encoding/binary"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database/lite"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/tests/helper"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

func TestInitiate(t *testing.T) {
	bus := wire.NewEventBus()
	keys, _ := user.NewRandKeys()
	rpcBus := wire.NewRPCBus()

	initChan := make(chan *bytes.Buffer, 1)
	bus.Subscribe(msg.InitializationTopic, initChan)

	if err := consensus.LaunchInitiator(bus, rpcBus); err != nil {
		t.Fatal(err)
	}

	bus.Publish(string(topics.StartConsensus), new(bytes.Buffer))
	// wait a bit for the initiator to receive the message
	time.Sleep(1 * time.Second)

	blk := helper.RandomBlock(t, 1, 2)
	stake := makeStake(keys)
	blk.AddTx(stake)

	buf := new(bytes.Buffer)
	if err := blk.Encode(buf); err != nil {
		t.Fatal(err)
	}

	bus.Publish(string(topics.AcceptedBlock), buf)

	roundBuf := <-initChan
	round := binary.LittleEndian.Uint64(roundBuf.Bytes())

	assert.Equal(t, uint64(2), round)
}

func setupDatabase() (database.Driver, database.DB) {
	drvr, err := database.From(lite.DriverName)
	if err != nil {
		panic(err)
	}

	db, err := drvr.Open("", protocol.TestNet, false)
	if err != nil {
		panic(err)
	}

	return drvr, db
}

func makeStake(keys user.Keys) *transactions.Stake {
	stake, _ := transactions.NewStake(0, math.MaxUint64, 100, *keys.EdPubKey, keys.BLSPubKeyBytes)
	keyImage, _ := crypto.RandEntropy(32)
	txID, _ := crypto.RandEntropy(32)
	signature, _ := crypto.RandEntropy(32)
	input, _ := transactions.NewInput(keyImage, txID, 0, signature)
	stake.Inputs = transactions.Inputs{input}

	outputAmount := rand.Int63n(100000)
	commitment := make([]byte, 32)
	binary.BigEndian.PutUint64(commitment[24:32], uint64(outputAmount))
	destKey, _ := crypto.RandEntropy(32)
	rangeProof, _ := crypto.RandEntropy(32)
	output, _ := transactions.NewOutput(commitment, destKey, rangeProof)
	stake.Outputs = transactions.Outputs{output}

	return stake
}
