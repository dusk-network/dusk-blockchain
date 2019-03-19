package selection_test

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/selection"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/prerror"
)

func TestSigSetCollection(t *testing.T) {
	verifyEd25519SignatureFunc := func(*bytes.Buffer) error {
		return nil
	}

	verifyVoteSetFunc := func([]*msg.Vote, []byte, uint64, uint8) *prerror.PrError {
		return nil
	}

	eventBus := wire.New()

	// subscribe to the outgoing topic
	selectionResultChannel := make(chan *bytes.Buffer, 1)
	eventBus.Subscribe(msg.SelectionResultTopic, selectionResultChannel)

	// subscribe to provisioneradded topic
	provisionerAddedChannel := make(chan *bytes.Buffer, 3)
	eventBus.Subscribe(msg.ProvisionerAddedTopic, provisionerAddedChannel)

	// Create a committee store
	committeeStore := committee.NewCommitteeStore(eventBus)
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
	setSelector := selection.NewSetSelector(eventBus, timerLength, verifyEd25519SignatureFunc,
		verifyVoteSetFunc, committeeStore)

	go setSelector.Listen()

	// initialise set selector
	roundBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(roundBytes[:8], 1)
	eventBus.Publish(msg.InitializationTopic, bytes.NewBuffer(roundBytes))

	// first, we set a winning block hash
	eventBus.Publish(string(topics.Agreement), bytes.NewBuffer(winningBlockHash))

	// Send messages
	eventBus.Publish(string(topics.SigSet), message1)
	eventBus.Publish(string(topics.SigSet), message2)
	eventBus.Publish(string(topics.SigSet), message3)

	// wait for a result from selectionResultChannel
	result := <-selectionResultChannel

	// result should be equal to vote set hash
	assert.Equal(t, voteSetHash, result.Bytes())

	// Kill goroutine
	eventBus.Publish(msg.QuitTopic, nil)
}

func TestInvalidVoteSetSigSetCollection(t *testing.T) {
	verifyEd25519SignatureFunc := func(*bytes.Buffer) error {
		return nil
	}

	verifyVoteSetFunc := func([]*msg.Vote, []byte, uint64, uint8) *prerror.PrError {
		return prerror.New(prerror.Low, errors.New("invalid vote set"))
	}

	eventBus := wire.New()

	// subscribe to the outgoing topic
	selectionResultChannel := make(chan *bytes.Buffer, 1)
	eventBus.Subscribe(msg.SelectionResultTopic, selectionResultChannel)

	// subscribe to provisioneradded topic
	provisionerAddedChannel := make(chan *bytes.Buffer, 3)
	eventBus.Subscribe(msg.ProvisionerAddedTopic, provisionerAddedChannel)

	// Create a committee store
	committeeStore := committee.NewCommitteeStore(eventBus)
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
	setSelector := selection.NewSetSelector(eventBus, timerLength, verifyEd25519SignatureFunc,
		verifyVoteSetFunc, committeeStore)

	go setSelector.Listen()

	// initialise set selector
	roundBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(roundBytes[:8], 1)
	eventBus.Publish(msg.InitializationTopic, bytes.NewBuffer(roundBytes))

	// first, we set a winning block hash
	eventBus.Publish(string(topics.Agreement), bytes.NewBuffer(winningBlockHash))

	// Send messages
	eventBus.Publish(string(topics.SigSet), message1)
	eventBus.Publish(string(topics.SigSet), message2)
	eventBus.Publish(string(topics.SigSet), message3)

	// wait for a result from selectionResultChannel
	result := <-selectionResultChannel

	// result should be nil
	assert.Nil(t, result.Bytes())

	// Kill goroutine
	eventBus.Publish(msg.QuitTopic, nil)
}

func TestInvalidSignatureSigSetCollection(t *testing.T) {
	verifyEd25519SignatureFunc := func(*bytes.Buffer) error {
		return errors.New("verification failed")
	}

	verifyVoteSetFunc := func([]*msg.Vote, []byte, uint64, uint8) *prerror.PrError {
		return nil
	}

	eventBus := wire.New()

	// subscribe to the outgoing topic
	selectionResultChannel := make(chan *bytes.Buffer, 1)
	eventBus.Subscribe(msg.SelectionResultTopic, selectionResultChannel)

	// subscribe to provisioneradded topic
	provisionerAddedChannel := make(chan *bytes.Buffer, 3)
	eventBus.Subscribe(msg.ProvisionerAddedTopic, provisionerAddedChannel)

	// Create a committee store
	committeeStore := committee.NewCommitteeStore(eventBus)
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
	setSelector := selection.NewSetSelector(eventBus, timerLength, verifyEd25519SignatureFunc,
		verifyVoteSetFunc, committeeStore)

	go setSelector.Listen()

	// initialise set selector
	roundBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(roundBytes[:8], 1)
	eventBus.Publish(msg.InitializationTopic, bytes.NewBuffer(roundBytes))

	// first, we set a winning block hash
	eventBus.Publish(string(topics.Agreement), bytes.NewBuffer(winningBlockHash))

	// Send messages
	eventBus.Publish(string(topics.SigSet), message1)
	eventBus.Publish(string(topics.SigSet), message2)
	eventBus.Publish(string(topics.SigSet), message3)

	// wait a bit...
	time.Sleep(200 * time.Millisecond)

	// selectionResultChannel should be empty
	assert.Empty(t, selectionResultChannel)

	// Kill goroutine
	eventBus.Publish(msg.QuitTopic, nil)
}

func TestSigSetCollectionQueue(t *testing.T) {
	verifyEd25519SignatureFunc := func(*bytes.Buffer) error {
		return nil
	}

	verifyVoteSetFunc := func([]*msg.Vote, []byte, uint64, uint8) *prerror.PrError {
		return nil
	}

	eventBus := wire.New()

	// subscribe to the outgoing topic
	selectionResultChannel := make(chan *bytes.Buffer, 1)
	eventBus.Subscribe(msg.SelectionResultTopic, selectionResultChannel)

	// subscribe to provisioneradded topic
	provisionerAddedChannel := make(chan *bytes.Buffer, 1)
	eventBus.Subscribe(msg.ProvisionerAddedTopic, provisionerAddedChannel)

	// Create a committee store
	committeeStore := committee.NewCommitteeStore(eventBus)
	go committeeStore.Listen()

	winningBlockHash, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	// make a vote set
	voteSet, err := newVoteSet(3, winningBlockHash, 1)
	if err != nil {
		t.Fatal(err)
	}

	// make a message for round 2
	message, voteSetHash, pubKeyBLS, err := newSigSetMessage(winningBlockHash,
		voteSet, 2, 1)
	if err != nil {
		t.Fatal(err)
	}

	// add a provisioner
	provisionerMessage, err := newProvisionerMessage(pubKeyBLS, 200)
	if err != nil {
		t.Fatal(err)
	}

	eventBus.Publish("addprovisioner", provisionerMessage)

	<-provisionerAddedChannel

	// Make a score selector with a short timeout
	timerLength := 100 * time.Millisecond
	setSelector := selection.NewSetSelector(eventBus, timerLength, verifyEd25519SignatureFunc,
		verifyVoteSetFunc, committeeStore)

	go setSelector.Listen()

	// initialise set selector
	roundBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(roundBytes[:8], 1)
	eventBus.Publish(msg.InitializationTopic, bytes.NewBuffer(roundBytes))

	// send message, should be queued
	eventBus.Publish(string(topics.SigSet), message)

	// increment round
	roundUpdateBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(roundUpdateBytes[:8], 2)
	eventBus.Publish(msg.RoundUpdateTopic, bytes.NewBuffer(roundUpdateBytes))

	// wait a moment...
	time.Sleep(100 * time.Millisecond)

	// set winning block hash
	eventBus.Publish(string(topics.Agreement), bytes.NewBuffer(winningBlockHash))

	// wait for a result from selectionResultChannel
	result := <-selectionResultChannel

	// result should be equal to vote set hash
	assert.Equal(t, voteSetHash, result.Bytes())

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
