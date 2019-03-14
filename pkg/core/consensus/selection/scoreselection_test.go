package selection_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"

	"github.com/stretchr/testify/assert"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/selection"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

func TestScoreCollection(t *testing.T) {
	validateFunc := func(*bytes.Buffer) error {
		return nil
	}

	verifyProofFunc := func([]byte, []byte, []byte, []byte, []byte) bool {
		return true
	}

	eventBus := wire.New()

	// subscribe to the outgoing topic
	outgoingChannel := make(chan *bytes.Buffer, 1)
	eventBus.Subscribe("outgoing", outgoingChannel)

	// Make a score selector with a short timeout
	timerLength := 100 * time.Millisecond
	scoreSelector := selection.NewScoreSelector(eventBus, timerLength, validateFunc,
		verifyProofFunc)

	go scoreSelector.Listen()

	// send three messages with different scores
	message1, _, err := newScoreMessage(500, 1, 1)
	if err != nil {
		t.Fatal(err)
	}

	eventBus.Publish("score", message1)
	message2, _, err := newScoreMessage(1500, 1, 1)
	if err != nil {
		t.Fatal(err)
	}

	eventBus.Publish("score", message2)
	message3, blockHash, err := newScoreMessage(5000, 1, 1)
	if err != nil {
		t.Fatal(err)
	}

	eventBus.Publish("score", message3)

	// wait for a result from outgoingChannel
	result := <-outgoingChannel

	// Check if it's the same as the block hash in message3
	assert.Equal(t, blockHash, result.Bytes())

	// Kill goroutine
	eventBus.Publish(msg.QuitTopic, nil)
}

func TestInvalidProofScoreCollection(t *testing.T) {
	validateFunc := func(*bytes.Buffer) error {
		return nil
	}

	verifyProofFunc := func([]byte, []byte, []byte, []byte, []byte) bool {
		return false
	}

	eventBus := wire.New()

	// subscribe to the outgoing topic
	outgoingChannel := make(chan *bytes.Buffer, 1)
	eventBus.Subscribe("outgoing", outgoingChannel)

	// Make a score selector with a short timeout
	timerLength := 100 * time.Millisecond
	scoreSelector := selection.NewScoreSelector(eventBus, timerLength, validateFunc,
		verifyProofFunc)

	go scoreSelector.Listen()

	// send three messages with different scores
	message1, _, err := newScoreMessage(500, 1, 1)
	if err != nil {
		t.Fatal(err)
	}

	eventBus.Publish("score", message1)
	message2, _, err := newScoreMessage(1500, 1, 1)
	if err != nil {
		t.Fatal(err)
	}

	eventBus.Publish("score", message2)
	message3, _, err := newScoreMessage(5000, 1, 1)
	if err != nil {
		t.Fatal(err)
	}

	eventBus.Publish("score", message3)

	// wait for a result from outgoingChannel
	result := <-outgoingChannel

	// Should be nil, as we got no proper messages
	assert.Nil(t, result.Bytes())

	// Kill goroutine
	eventBus.Publish(msg.QuitTopic, nil)
}

func TestInvalidSignatureScoreCollection(t *testing.T) {
	validateFunc := func(*bytes.Buffer) error {
		return errors.New("verification failed")
	}

	verifyProofFunc := func([]byte, []byte, []byte, []byte, []byte) bool {
		return true
	}

	eventBus := wire.New()

	// subscribe to the outgoing topic
	outgoingChannel := make(chan *bytes.Buffer, 1)
	eventBus.Subscribe("outgoing", outgoingChannel)

	// Make a score selector with a short timeout
	timerLength := 100 * time.Millisecond
	scoreSelector := selection.NewScoreSelector(eventBus, timerLength, validateFunc,
		verifyProofFunc)

	go scoreSelector.Listen()

	// send three messages with different scores
	message1, _, err := newScoreMessage(500, 1, 1)
	if err != nil {
		t.Fatal(err)
	}

	eventBus.Publish("score", message1)
	message2, _, err := newScoreMessage(1500, 1, 1)
	if err != nil {
		t.Fatal(err)
	}

	eventBus.Publish("score", message2)
	message3, _, err := newScoreMessage(5000, 1, 1)
	if err != nil {
		t.Fatal(err)
	}

	eventBus.Publish("score", message3)

	// wait a bit...
	time.Sleep(200 * time.Millisecond)

	// outgoingChannel should be empty, as the collection round should not
	// have started
	assert.Empty(t, outgoingChannel)

	// Kill goroutine
	eventBus.Publish(msg.QuitTopic, nil)
}

func TestScoreCollectionQueue(t *testing.T) {
	validateFunc := func(*bytes.Buffer) error {
		return nil
	}

	verifyProofFunc := func([]byte, []byte, []byte, []byte, []byte) bool {
		return true
	}

	eventBus := wire.New()

	// subscribe to the outgoing topic
	outgoingChannel := make(chan *bytes.Buffer, 1)
	eventBus.Subscribe("outgoing", outgoingChannel)

	// Make a score selector with a short timeout
	timerLength := 100 * time.Millisecond
	scoreSelector := selection.NewScoreSelector(eventBus, timerLength, validateFunc,
		verifyProofFunc)

	go scoreSelector.Listen()

	// send one message to set off collection, and initialise our round
	// and step
	message1, blockHash1, err := newScoreMessage(500, 1, 1)
	if err != nil {
		t.Fatal(err)
	}

	eventBus.Publish("score", message1)

	// wait for a result from outgoingChannel
	result1 := <-outgoingChannel

	// we should have gotten blockHash1 from outgoingChannel
	assert.Equal(t, blockHash1, result1.Bytes())

	// send two messages one round ahead, which should be stored
	message2, _, err := newScoreMessage(1500, 2, 1)
	if err != nil {
		t.Fatal(err)
	}

	eventBus.Publish("score", message2)
	message3, blockHash2, err := newScoreMessage(5000, 2, 1)
	if err != nil {
		t.Fatal(err)
	}

	eventBus.Publish("score", message3)

	// the queue should now hold messages on round 2, step 1
	// we will increment the round, which should cause the other messages
	// to be retrieved, and another collection round should start
	roundBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(roundBytes, 2)
	eventBus.Publish("roundupdate", bytes.NewBuffer(roundBytes))

	// wait for another result from outgoingChannel
	result2 := <-outgoingChannel

	// we should have gotten blockHash2 from outgoingChannel
	assert.Equal(t, blockHash2, result2.Bytes())

	// Kill goroutine
	eventBus.Publish(msg.QuitTopic, nil)
}

func newScoreMessage(score, round uint64, step uint8) (*bytes.Buffer, []byte, error) {
	buffer := new(bytes.Buffer)

	scoreBytes := make([]byte, 32)
	binary.BigEndian.PutUint64(scoreBytes[24:32], score)
	if err := encoding.Write256(buffer, scoreBytes); err != nil {
		return nil, nil, err
	}

	proof, err := crypto.RandEntropy(100)
	if err != nil {
		return nil, nil, err
	}

	if err := encoding.WriteVarBytes(buffer, proof); err != nil {
		return nil, nil, err
	}

	byte32, err := crypto.RandEntropy(32)
	if err != nil {
		return nil, nil, err
	}

	// Z
	if err := encoding.Write256(buffer, byte32); err != nil {
		return nil, nil, err
	}

	bidListSubset, err := crypto.RandEntropy(0)
	if err != nil {
		return nil, nil, err
	}

	if err := encoding.WriteVarBytes(buffer, bidListSubset); err != nil {
		return nil, nil, err
	}

	seed, err := crypto.RandEntropy(33)
	if err != nil {
		return nil, nil, err
	}

	if err := encoding.WriteBLS(buffer, seed); err != nil {
		return nil, nil, err
	}

	// CandidateHash
	if err := encoding.Write256(buffer, byte32); err != nil {
		return nil, nil, err
	}

	// Round
	if err := encoding.WriteUint64(buffer, binary.LittleEndian, round); err != nil {
		return nil, nil, err
	}

	// Step
	if err := encoding.WriteUint8(buffer, step); err != nil {
		return nil, nil, err
	}

	return buffer, byte32, nil
}
