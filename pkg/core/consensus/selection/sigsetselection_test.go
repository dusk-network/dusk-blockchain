package selection_test

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/commands"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/prerror"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/selection"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

func TestSigSetCollection(t *testing.T) {
	validateFunc := func(*bytes.Buffer) error {
		return nil
	}

	verifyVoteFunc := func(*msg.Vote, []byte, uint8, map[string]uint8,
		map[string]uint8) *prerror.PrError {
		return nil
	}

	eventBus := wire.New()

	// subscribe to the outgoing topic
	outgoingChannel := make(chan *bytes.Buffer, 1)
	eventBus.Subscribe("outgoing", outgoingChannel)

	// subscribe to provisioneradded topic
	provisionerAddedChannel := make(chan *bytes.Buffer, 3)
	eventBus.Subscribe(msg.ProvisionerAddedTopic, provisionerAddedChannel)

	// Create a committee store
	committeeStore := user.NewCommitteeStore(eventBus)
	go committeeStore.Listen()

	winningBlockHash, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	// make a common vote set
	voteSet, err := newVoteSet(3, winningBlockHash, 1)
	if err != nil {
		t.Fatal(err)
	}

	// make three messages
	message1, voteSetHash, pubKeyBLS1, err := newSigSetMessage(winningBlockHash,
		voteSet, 1, 1)
	if err != nil {
		t.Fatal(err)
	}

	message2, _, pubKeyBLS2, err := newSigSetMessage(winningBlockHash,
		voteSet, 1, 1)
	if err != nil {
		t.Fatal(err)
	}

	message3, _, pubKeyBLS3, err := newSigSetMessage(winningBlockHash,
		voteSet, 1, 1)
	if err != nil {
		t.Fatal(err)
	}

	// add three provisioners
	provisionerMessage1, err := newProvisionerMessage(pubKeyBLS1, 200)
	if err != nil {
		t.Fatal(err)
	}

	provisionerMessage2, err := newProvisionerMessage(pubKeyBLS2, 400)
	if err != nil {
		t.Fatal(err)
	}

	provisionerMessage3, err := newProvisionerMessage(pubKeyBLS3, 1000)
	if err != nil {
		t.Fatal(err)
	}

	eventBus.Publish("addprovisioner", provisionerMessage1)
	eventBus.Publish("addprovisioner", provisionerMessage2)
	eventBus.Publish("addprovisioner", provisionerMessage3)

	<-provisionerAddedChannel
	<-provisionerAddedChannel
	<-provisionerAddedChannel

	// Make a score selector with a short timeout
	timerLength := 100 * time.Millisecond
	setSelector := selection.NewSetSelector(eventBus, timerLength, validateFunc,
		verifyVoteFunc, committeeStore)

	go setSelector.Listen()

	// first, we set a winning block hash
	eventBus.Publish(string(commands.Agreement), bytes.NewBuffer(winningBlockHash))

	// Send messages
	eventBus.Publish("sigset", message1)
	eventBus.Publish("sigset", message2)
	eventBus.Publish("sigset", message3)

	// wait for a result from outgoingChannel
	result := <-outgoingChannel

	// result should be equal to vote set hash
	assert.Equal(t, voteSetHash, result.Bytes())

	// Kill goroutine
	eventBus.Publish(msg.QuitTopic, nil)
}

func TestInvalidVoteSetSigSetCollection(t *testing.T) {
	validateFunc := func(*bytes.Buffer) error {
		return nil
	}

	verifyVoteFunc := func(*msg.Vote, []byte, uint8, map[string]uint8,
		map[string]uint8) *prerror.PrError {
		return prerror.New(prerror.Low, errors.New("invalid vote set"))
	}

	eventBus := wire.New()

	// subscribe to the outgoing topic
	outgoingChannel := make(chan *bytes.Buffer, 1)
	eventBus.Subscribe("outgoing", outgoingChannel)

	// subscribe to provisioneradded topic
	provisionerAddedChannel := make(chan *bytes.Buffer, 3)
	eventBus.Subscribe(msg.ProvisionerAddedTopic, provisionerAddedChannel)

	// Create a committee store
	committeeStore := user.NewCommitteeStore(eventBus)
	go committeeStore.Listen()

	winningBlockHash, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	// make a common vote set
	voteSet, err := newVoteSet(3, winningBlockHash, 1)
	if err != nil {
		t.Fatal(err)
	}

	// make three messages
	message1, _, pubKeyBLS1, err := newSigSetMessage(winningBlockHash,
		voteSet, 1, 1)
	if err != nil {
		t.Fatal(err)
	}

	message2, _, pubKeyBLS2, err := newSigSetMessage(winningBlockHash,
		voteSet, 1, 1)
	if err != nil {
		t.Fatal(err)
	}

	message3, _, pubKeyBLS3, err := newSigSetMessage(winningBlockHash,
		voteSet, 1, 1)
	if err != nil {
		t.Fatal(err)
	}

	// add three provisioners
	provisionerMessage1, err := newProvisionerMessage(pubKeyBLS1, 200)
	if err != nil {
		t.Fatal(err)
	}

	provisionerMessage2, err := newProvisionerMessage(pubKeyBLS2, 400)
	if err != nil {
		t.Fatal(err)
	}

	provisionerMessage3, err := newProvisionerMessage(pubKeyBLS3, 1000)
	if err != nil {
		t.Fatal(err)
	}

	eventBus.Publish("addprovisioner", provisionerMessage1)
	eventBus.Publish("addprovisioner", provisionerMessage2)
	eventBus.Publish("addprovisioner", provisionerMessage3)

	<-provisionerAddedChannel
	<-provisionerAddedChannel
	<-provisionerAddedChannel

	// Make a score selector with a short timeout
	timerLength := 100 * time.Millisecond
	setSelector := selection.NewSetSelector(eventBus, timerLength, validateFunc,
		verifyVoteFunc, committeeStore)

	go setSelector.Listen()

	// first, we set a winning block hash
	eventBus.Publish(string(commands.Agreement), bytes.NewBuffer(winningBlockHash))

	// Send messages
	eventBus.Publish("sigset", message1)
	eventBus.Publish("sigset", message2)
	eventBus.Publish("sigset", message3)

	// wait for a result from outgoingChannel
	result := <-outgoingChannel

	// result should be nil
	assert.Nil(t, result.Bytes())

	// Kill goroutine
	eventBus.Publish(msg.QuitTopic, nil)
}

func TestInvalidSignatureSigSetCollection(t *testing.T) {
	validateFunc := func(*bytes.Buffer) error {
		return errors.New("verification failed")
	}

	verifyVoteFunc := func(*msg.Vote, []byte, uint8, map[string]uint8,
		map[string]uint8) *prerror.PrError {
		return nil
	}

	eventBus := wire.New()

	// subscribe to the outgoing topic
	outgoingChannel := make(chan *bytes.Buffer, 1)
	eventBus.Subscribe("outgoing", outgoingChannel)

	// subscribe to provisioneradded topic
	provisionerAddedChannel := make(chan *bytes.Buffer, 3)
	eventBus.Subscribe(msg.ProvisionerAddedTopic, provisionerAddedChannel)

	// Create a committee store
	committeeStore := user.NewCommitteeStore(eventBus)
	go committeeStore.Listen()

	winningBlockHash, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	// make a common vote set
	voteSet, err := newVoteSet(3, winningBlockHash, 1)
	if err != nil {
		t.Fatal(err)
	}

	// make three messages
	message1, _, pubKeyBLS1, err := newSigSetMessage(winningBlockHash,
		voteSet, 1, 1)
	if err != nil {
		t.Fatal(err)
	}

	message2, _, pubKeyBLS2, err := newSigSetMessage(winningBlockHash,
		voteSet, 1, 1)
	if err != nil {
		t.Fatal(err)
	}

	message3, _, pubKeyBLS3, err := newSigSetMessage(winningBlockHash,
		voteSet, 1, 1)
	if err != nil {
		t.Fatal(err)
	}

	// add three provisioners
	provisionerMessage1, err := newProvisionerMessage(pubKeyBLS1, 200)
	if err != nil {
		t.Fatal(err)
	}

	provisionerMessage2, err := newProvisionerMessage(pubKeyBLS2, 400)
	if err != nil {
		t.Fatal(err)
	}

	provisionerMessage3, err := newProvisionerMessage(pubKeyBLS3, 1000)
	if err != nil {
		t.Fatal(err)
	}

	eventBus.Publish("addprovisioner", provisionerMessage1)
	eventBus.Publish("addprovisioner", provisionerMessage2)
	eventBus.Publish("addprovisioner", provisionerMessage3)

	<-provisionerAddedChannel
	<-provisionerAddedChannel
	<-provisionerAddedChannel

	// Make a score selector with a short timeout
	timerLength := 100 * time.Millisecond
	setSelector := selection.NewSetSelector(eventBus, timerLength, validateFunc,
		verifyVoteFunc, committeeStore)

	go setSelector.Listen()

	// first, we set a winning block hash
	eventBus.Publish(string(commands.Agreement), bytes.NewBuffer(winningBlockHash))

	// Send messages
	eventBus.Publish("sigset", message1)
	eventBus.Publish("sigset", message2)
	eventBus.Publish("sigset", message3)

	// wait a bit...
	time.Sleep(200 * time.Millisecond)

	// outgoingChannel should be empty
	assert.Empty(t, outgoingChannel)

	// Kill goroutine
	eventBus.Publish(msg.QuitTopic, nil)
}

func TestSigSetCollectionQueue(t *testing.T) {
	validateFunc := func(*bytes.Buffer) error {
		return nil
	}

	verifyVoteFunc := func(*msg.Vote, []byte, uint8, map[string]uint8,
		map[string]uint8) *prerror.PrError {
		return nil
	}

	eventBus := wire.New()

	// subscribe to the outgoing topic
	outgoingChannel := make(chan *bytes.Buffer, 1)
	eventBus.Subscribe("outgoing", outgoingChannel)

	// subscribe to provisioneradded topic
	provisionerAddedChannel := make(chan *bytes.Buffer, 3)
	eventBus.Subscribe(msg.ProvisionerAddedTopic, provisionerAddedChannel)

	// Create a committee store
	committeeStore := user.NewCommitteeStore(eventBus)
	go committeeStore.Listen()

	winningBlockHash, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	// make a vote set for round 1
	voteSet1, err := newVoteSet(3, winningBlockHash, 1)
	if err != nil {
		t.Fatal(err)
	}

	// make a vote set for round 2
	voteSet2, err := newVoteSet(3, winningBlockHash, 1)
	if err != nil {
		t.Fatal(err)
	}

	// make a message for each round
	message1, voteSetHash1, pubKeyBLS1, err := newSigSetMessage(winningBlockHash,
		voteSet1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}

	message2, voteSetHash2, pubKeyBLS2, err := newSigSetMessage(winningBlockHash,
		voteSet2, 2, 1)
	if err != nil {
		t.Fatal(err)
	}

	// add two provisioners
	provisionerMessage1, err := newProvisionerMessage(pubKeyBLS1, 200)
	if err != nil {
		t.Fatal(err)
	}

	// give the second provisioner a huge chance of being included
	provisionerMessage2, err := newProvisionerMessage(pubKeyBLS2, 100000)
	if err != nil {
		t.Fatal(err)
	}

	eventBus.Publish("addprovisioner", provisionerMessage1)

	<-provisionerAddedChannel

	// Make a score selector with a short timeout
	timerLength := 100 * time.Millisecond
	setSelector := selection.NewSetSelector(eventBus, timerLength, validateFunc,
		verifyVoteFunc, committeeStore)

	go setSelector.Listen()

	// first, we set a winning block hash
	eventBus.Publish(string(commands.Agreement), bytes.NewBuffer(winningBlockHash))

	// send message, which should give us a result identical to voteSetHash1
	eventBus.Publish("sigset", message1)

	// wait for a result from outgoingChannel
	result1 := <-outgoingChannel

	// result should be equal to vote set hash 1
	assert.Equal(t, voteSetHash1, result1.Bytes())

	// send other messages
	eventBus.Publish("addprovisioner", provisionerMessage2)

	<-provisionerAddedChannel

	eventBus.Publish("sigset", message2)

	// increment round
	roundBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(roundBytes[:8], 2)
	eventBus.Publish(msg.RoundUpdateTopic, bytes.NewBuffer(roundBytes))

	// set winning block hash again
	eventBus.Publish(string(commands.Agreement), bytes.NewBuffer(winningBlockHash))

	// wait for a result from outgoingChannel
	result2 := <-outgoingChannel

	// result should be equal to vote set hash 2
	assert.Equal(t, voteSetHash2, result2.Bytes())

	// Kill goroutine
	eventBus.Publish(msg.QuitTopic, nil)
}

func newVoteSet(amount int, hash []byte, step uint8) ([]*msg.Vote, error) {
	var voteSet []*msg.Vote
	for i := 0; i < amount; i++ {
		pubKeyBLS, privKeyBLS, err := bls.GenKeyPair(rand.Reader)
		if err != nil {
			return nil, err
		}

		sig, err := bls.Sign(privKeyBLS, pubKeyBLS, hash)
		if err != nil {
			return nil, err
		}

		compressedSig := sig.Compress()

		voteSet = append(voteSet, &msg.Vote{
			VotedHash:  hash,
			PubKeyBLS:  pubKeyBLS.Marshal(),
			SignedHash: compressedSig,
			Step:       step,
		})
	}

	for i := 0; i < amount; i++ {
		pubKeyBLS, privKeyBLS, err := bls.GenKeyPair(rand.Reader)
		if err != nil {
			return nil, err
		}

		sig, err := bls.Sign(privKeyBLS, pubKeyBLS, hash)
		if err != nil {
			return nil, err
		}

		compressedSig := sig.Compress()

		voteSet = append(voteSet, &msg.Vote{
			VotedHash:  hash,
			PubKeyBLS:  pubKeyBLS.Marshal(),
			SignedHash: compressedSig,
			Step:       step - 1,
		})
	}

	return voteSet, nil
}

func newProvisionerMessage(pubKeyBLS []byte, amount uint64) (*bytes.Buffer, error) {
	buffer := new(bytes.Buffer)
	pubKeyEd, err := crypto.RandEntropy(32)
	if err != nil {
		return nil, err
	}

	if err := encoding.Write256(buffer, pubKeyEd); err != nil {
		return nil, err
	}

	if err := encoding.WriteVarBytes(buffer, pubKeyBLS); err != nil {
		return nil, err
	}

	if err := encoding.WriteUint64(buffer, binary.LittleEndian, amount); err != nil {
		return nil, err
	}

	return buffer, nil
}

func newSigSetMessage(winningBlockHash []byte, voteSet []*msg.Vote, round uint64,
	step uint8) (*bytes.Buffer, []byte, []byte, error) {

	buffer := new(bytes.Buffer)
	pubKeyBLS, privKeyBLS, err := bls.GenKeyPair(rand.Reader)
	if err != nil {
		return nil, nil, nil, err
	}

	if err := encoding.Write256(buffer, winningBlockHash); err != nil {
		return nil, nil, nil, err
	}

	voteSetBytes, err := msg.EncodeVoteSet(voteSet)
	if err != nil {
		return nil, nil, nil, err
	}

	buffer.Write(voteSetBytes)

	hashedVoteSet, err := msg.HashVoteSet(voteSet)
	if err != nil {
		return nil, nil, nil, err
	}

	signedVoteSet, err := bls.Sign(privKeyBLS, pubKeyBLS, hashedVoteSet)
	if err != nil {
		return nil, nil, nil, err
	}

	if err := encoding.WriteBLS(buffer, signedVoteSet.Compress()); err != nil {
		return nil, nil, nil, err
	}

	if err := encoding.WriteVarBytes(buffer, pubKeyBLS.Marshal()); err != nil {
		return nil, nil, nil, err
	}

	if err := encoding.WriteUint64(buffer, binary.LittleEndian, round); err != nil {
		return nil, nil, nil, err
	}

	if err := encoding.WriteUint8(buffer, step); err != nil {
		return nil, nil, nil, err
	}

	return buffer, hashedVoteSet, pubKeyBLS.Marshal(), nil
}
