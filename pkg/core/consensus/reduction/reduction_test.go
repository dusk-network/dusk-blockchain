package reduction_test

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/reduction"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

func TestBlockReduction(t *testing.T) {
	verifyEd25519SignatureFunc := func(*bytes.Buffer) error {
		return nil
	}

	eventBus := wire.New()

	// subscribe to the reductionresult topic
	reductionResultChannel := make(chan *bytes.Buffer, 3)
	eventBus.Subscribe(msg.ReductionResultTopic, reductionResultChannel)

	// subscribe to provisioneradded topic
	provisionerAddedChannel := make(chan *bytes.Buffer, 3)
	eventBus.Subscribe(msg.ProvisionerAddedTopic, provisionerAddedChannel)

	// Create a committee store
	committeeStore := user.NewCommitteeStore(eventBus)
	go committeeStore.Listen()

	// make three bls key pairs
	pubKeyBLS1, privKeyBLS1, err := bls.GenKeyPair(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	pubKeyBLS2, privKeyBLS2, err := bls.GenKeyPair(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	pubKeyBLS3, privKeyBLS3, err := bls.GenKeyPair(rand.Reader)
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

	// make block hash
	blockHash, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	// make three messages
	message1, err := newBlockReductionMessage(blockHash, 1, 1, pubKeyBLS1, privKeyBLS1)
	if err != nil {
		t.Fatal(err)
	}

	message2, err := newBlockReductionMessage(blockHash, 1, 1, pubKeyBLS2, privKeyBLS2)
	if err != nil {
		t.Fatal(err)
	}

	message3, err := newBlockReductionMessage(blockHash, 1, 1, pubKeyBLS3, privKeyBLS3)
	if err != nil {
		t.Fatal(err)
	}

	// Set up reducer with a short timeout
	timerLength := 100 * time.Millisecond
	keys, err := user.NewRandKeys()
	if err != nil {
		t.Fatal(err)
	}

	reducer := reduction.NewReducer(eventBus, timerLength, verifyEd25519SignatureFunc,
		committeeStore, keys)

	go reducer.Listen()

	// initialise reducer
	roundBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(roundBytes[:8], 1)
	eventBus.Publish(msg.InitializationTopic, bytes.NewBuffer(roundBytes))

	// now send the messages and kick off a reduction cycle
	eventBus.Publish(string(topics.BlockReduction), message1)
	eventBus.Publish(string(topics.BlockReduction), message2)
	eventBus.Publish(string(topics.BlockReduction), message3)

	// create three messages for the next step
	message4, err := newBlockReductionMessage(blockHash, 1, 2, pubKeyBLS1, privKeyBLS1)
	if err != nil {
		t.Fatal(err)
	}

	message5, err := newBlockReductionMessage(blockHash, 1, 2, pubKeyBLS2, privKeyBLS2)
	if err != nil {
		t.Fatal(err)
	}

	message6, err := newBlockReductionMessage(blockHash, 1, 2, pubKeyBLS3, privKeyBLS3)
	if err != nil {
		t.Fatal(err)
	}

	// send the messages
	eventBus.Publish(string(topics.BlockReduction), message4)
	eventBus.Publish(string(topics.BlockReduction), message5)
	eventBus.Publish(string(topics.BlockReduction), message6)

	// check if we got a proper result
	m := <-reductionResultChannel
	assert.Equal(t, blockHash, m.Bytes())

	// Kill goroutine
	eventBus.Publish(msg.QuitTopic, nil)
}

func TestSigSetReduction(t *testing.T) {
	verifyEd25519SignatureFunc := func(*bytes.Buffer) error {
		return nil
	}

	eventBus := wire.New()

	// subscribe to the reductionresult topic
	reductionResultChannel := make(chan *bytes.Buffer, 3)
	eventBus.Subscribe(msg.ReductionResultTopic, reductionResultChannel)

	// subscribe to provisioneradded topic
	provisionerAddedChannel := make(chan *bytes.Buffer, 3)
	eventBus.Subscribe(msg.ProvisionerAddedTopic, provisionerAddedChannel)

	// Create a committee store
	committeeStore := user.NewCommitteeStore(eventBus)
	go committeeStore.Listen()

	// make three bls key pairs
	pubKeyBLS1, privKeyBLS1, err := bls.GenKeyPair(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	pubKeyBLS2, privKeyBLS2, err := bls.GenKeyPair(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	pubKeyBLS3, privKeyBLS3, err := bls.GenKeyPair(rand.Reader)
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

	// make winning block hash
	winningBlockHash, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	// make sig set hash
	sigSetHash, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	// make three messages
	message1, err := newSigSetReductionMessage(sigSetHash, winningBlockHash, 1, 1,
		pubKeyBLS1, privKeyBLS1)
	if err != nil {
		t.Fatal(err)
	}

	message2, err := newSigSetReductionMessage(sigSetHash, winningBlockHash, 1, 1,
		pubKeyBLS2, privKeyBLS2)
	if err != nil {
		t.Fatal(err)
	}

	message3, err := newSigSetReductionMessage(sigSetHash, winningBlockHash, 1, 1,
		pubKeyBLS3, privKeyBLS3)
	if err != nil {
		t.Fatal(err)
	}

	// Set up reducer with a short timeout
	timerLength := 100 * time.Millisecond
	keys, err := user.NewRandKeys()
	if err != nil {
		t.Fatal(err)
	}

	reducer := reduction.NewReducer(eventBus, timerLength, verifyEd25519SignatureFunc,
		committeeStore, keys)

	go reducer.Listen()

	// initialise reducer
	roundBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(roundBytes[:8], 1)
	eventBus.Publish(msg.InitializationTopic, bytes.NewBuffer(roundBytes))

	// send winning block hash to go into sig set phase
	eventBus.Publish(string(topics.Agreement), bytes.NewBuffer(winningBlockHash))

	// now send the messages and kick off a reduction cycle
	eventBus.Publish(string(topics.SigSetReduction), message1)
	eventBus.Publish(string(topics.SigSetReduction), message2)
	eventBus.Publish(string(topics.SigSetReduction), message3)

	// create three messages for the next step
	message4, err := newSigSetReductionMessage(sigSetHash, winningBlockHash, 1, 2,
		pubKeyBLS1, privKeyBLS1)
	if err != nil {
		t.Fatal(err)
	}

	message5, err := newSigSetReductionMessage(sigSetHash, winningBlockHash, 1, 2,
		pubKeyBLS2, privKeyBLS2)
	if err != nil {
		t.Fatal(err)
	}

	message6, err := newSigSetReductionMessage(sigSetHash, winningBlockHash, 1, 2,
		pubKeyBLS3, privKeyBLS3)
	if err != nil {
		t.Fatal(err)
	}

	// send the messages
	eventBus.Publish(string(topics.SigSetReduction), message4)
	eventBus.Publish(string(topics.SigSetReduction), message5)
	eventBus.Publish(string(topics.SigSetReduction), message6)

	// check if we got a proper result
	m := <-reductionResultChannel
	assert.Equal(t, sigSetHash, m.Bytes())

	// Kill goroutine
	eventBus.Publish(msg.QuitTopic, nil)
}

func TestInvalidSignatureReduction(t *testing.T) {
	verifyEd25519SignatureFunc := func(*bytes.Buffer) error {
		return errors.New("verification failed")
	}

	eventBus := wire.New()

	// subscribe to the reductionresult topic
	reductionResultChannel := make(chan *bytes.Buffer, 3)
	eventBus.Subscribe(msg.ReductionResultTopic, reductionResultChannel)

	// subscribe to provisioneradded topic
	provisionerAddedChannel := make(chan *bytes.Buffer, 3)
	eventBus.Subscribe(msg.ProvisionerAddedTopic, provisionerAddedChannel)

	// Create a committee store
	committeeStore := user.NewCommitteeStore(eventBus)
	go committeeStore.Listen()

	// make three bls key pairs
	pubKeyBLS1, privKeyBLS1, err := bls.GenKeyPair(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	pubKeyBLS2, privKeyBLS2, err := bls.GenKeyPair(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	pubKeyBLS3, privKeyBLS3, err := bls.GenKeyPair(rand.Reader)
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

	// make block hash
	blockHash, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	// make three messages
	message1, err := newBlockReductionMessage(blockHash, 1, 1, pubKeyBLS1, privKeyBLS1)
	if err != nil {
		t.Fatal(err)
	}

	message2, err := newBlockReductionMessage(blockHash, 1, 1, pubKeyBLS2, privKeyBLS2)
	if err != nil {
		t.Fatal(err)
	}

	message3, err := newBlockReductionMessage(blockHash, 1, 1, pubKeyBLS3, privKeyBLS3)
	if err != nil {
		t.Fatal(err)
	}

	// Set up reducer with a short timeout
	timerLength := 100 * time.Millisecond
	keys, err := user.NewRandKeys()
	if err != nil {
		t.Fatal(err)
	}

	reducer := reduction.NewReducer(eventBus, timerLength, verifyEd25519SignatureFunc,
		committeeStore, keys)

	go reducer.Listen()

	// initialise reducer
	roundBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(roundBytes[:8], 1)
	eventBus.Publish(msg.InitializationTopic, bytes.NewBuffer(roundBytes))

	// now send the messages and kick off a reduction cycle
	eventBus.Publish(string(topics.BlockReduction), message1)
	eventBus.Publish(string(topics.BlockReduction), message2)
	eventBus.Publish(string(topics.BlockReduction), message3)

	// create three messages for the next step
	message4, err := newBlockReductionMessage(blockHash, 1, 2, pubKeyBLS1, privKeyBLS1)
	if err != nil {
		t.Fatal(err)
	}

	message5, err := newBlockReductionMessage(blockHash, 1, 2, pubKeyBLS2, privKeyBLS2)
	if err != nil {
		t.Fatal(err)
	}

	message6, err := newBlockReductionMessage(blockHash, 1, 2, pubKeyBLS3, privKeyBLS3)
	if err != nil {
		t.Fatal(err)
	}

	// send the messages
	eventBus.Publish(string(topics.BlockReduction), message4)
	eventBus.Publish(string(topics.BlockReduction), message5)
	eventBus.Publish(string(topics.BlockReduction), message6)

	// wait a bit...
	time.Sleep(200 * time.Millisecond)

	// result channel should be empty
	assert.Empty(t, reductionResultChannel)

	// Kill goroutine
	eventBus.Publish(msg.QuitTopic, nil)
}

func TestBlockReductionVoting(t *testing.T) {
	verifyEd25519SignatureFunc := func(*bytes.Buffer) error {
		return nil
	}

	eventBus := wire.New()

	// subscribe to the reductionresult topic
	reductionResultChannel := make(chan *bytes.Buffer, 3)
	eventBus.Subscribe(msg.ReductionResultTopic, reductionResultChannel)

	// subscribe to outgoingreduction topic
	outgoingReductionChannel := make(chan *bytes.Buffer, 10)
	eventBus.Subscribe(msg.OutgoingReductionTopic, outgoingReductionChannel)

	// subscribe to outgoingagreement topic
	outgoingAgreementChannel := make(chan *bytes.Buffer, 10)
	eventBus.Subscribe(msg.OutgoingAgreementTopic, outgoingAgreementChannel)

	// subscribe to provisioneradded topic
	provisionerAddedChannel := make(chan *bytes.Buffer, 3)
	eventBus.Subscribe(msg.ProvisionerAddedTopic, provisionerAddedChannel)

	// Create a committee store
	committeeStore := user.NewCommitteeStore(eventBus)
	go committeeStore.Listen()

	// Set up reducer with a short timeout
	timerLength := 100 * time.Millisecond
	keys, err := user.NewRandKeys()
	if err != nil {
		t.Fatal(err)
	}

	reducer := reduction.NewReducer(eventBus, timerLength, verifyEd25519SignatureFunc,
		committeeStore, keys)

	go reducer.Listen()

	// Add only the reducer as provisioner
	provisionerMessage, err := newProvisionerMessage(reducer.BLSPubKey, 500)
	if err != nil {
		t.Fatal(err)
	}

	eventBus.Publish("addprovisioner", provisionerMessage)

	<-provisionerAddedChannel

	// initialise reducer
	roundBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(roundBytes[:8], 1)
	eventBus.Publish(msg.InitializationTopic, bytes.NewBuffer(roundBytes))

	// make block hash
	blockHash, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	// make messages
	message1, err := newBlockReductionMessage(blockHash, 1, 1, reducer.BLSPubKey,
		reducer.BLSSecretKey)
	if err != nil {
		t.Fatal(err)
	}

	message2, err := newBlockReductionMessage(blockHash, 1, 2, reducer.BLSPubKey,
		reducer.BLSSecretKey)
	if err != nil {
		t.Fatal(err)
	}

	// send a selection result to trigger the first vote
	eventBus.Publish(msg.SelectionResultTopic, bytes.NewBuffer(blockHash))

	// get first vote
	<-outgoingReductionChannel

	// now send the messages and kick off a reduction cycle
	eventBus.Publish(string(topics.BlockReduction), message1)
	eventBus.Publish(string(topics.BlockReduction), message2)

	// wait for result
	<-reductionResultChannel

	// we should have a reduction vote in our outgoingreduction channel
	assert.NotEmpty(t, outgoingReductionChannel)

	// we should have a message in our outgoingagreementchannel
	assert.NotEmpty(t, outgoingAgreementChannel)

	// Kill goroutine
	eventBus.Publish(msg.QuitTopic, nil)
}

func TestSigSetReductionVoting(t *testing.T) {
	verifyEd25519SignatureFunc := func(*bytes.Buffer) error {
		return nil
	}

	eventBus := wire.New()

	// subscribe to the reductionresult topic
	reductionResultChannel := make(chan *bytes.Buffer, 3)
	eventBus.Subscribe(msg.ReductionResultTopic, reductionResultChannel)

	// subscribe to outgoingreduction topic
	outgoingReductionChannel := make(chan *bytes.Buffer, 10)
	eventBus.Subscribe(msg.OutgoingReductionTopic, outgoingReductionChannel)

	// subscribe to outgoingagreement topic
	outgoingAgreementChannel := make(chan *bytes.Buffer, 10)
	eventBus.Subscribe(msg.OutgoingAgreementTopic, outgoingAgreementChannel)

	// subscribe to provisioneradded topic
	provisionerAddedChannel := make(chan *bytes.Buffer, 3)
	eventBus.Subscribe(msg.ProvisionerAddedTopic, provisionerAddedChannel)

	// Create a committee store
	committeeStore := user.NewCommitteeStore(eventBus)
	go committeeStore.Listen()

	// Set up reducer with a short timeout
	timerLength := 100 * time.Millisecond
	keys, err := user.NewRandKeys()
	if err != nil {
		t.Fatal(err)
	}

	reducer := reduction.NewReducer(eventBus, timerLength, verifyEd25519SignatureFunc,
		committeeStore, keys)

	go reducer.Listen()

	// Add only the reducer as provisioner
	provisionerMessage, err := newProvisionerMessage(reducer.BLSPubKey, 500)
	if err != nil {
		t.Fatal(err)
	}

	eventBus.Publish("addprovisioner", provisionerMessage)

	<-provisionerAddedChannel

	// initialise reducer
	roundBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(roundBytes[:8], 1)
	eventBus.Publish(msg.InitializationTopic, bytes.NewBuffer(roundBytes))

	// make sig set hash
	sigSetHash, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	// make block hash
	winningBlockHash, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	// set to sig set phase
	eventBus.Publish(string(topics.Agreement), bytes.NewBuffer(winningBlockHash))

	// make messages
	message1, err := newSigSetReductionMessage(sigSetHash, winningBlockHash, 1, 1,
		reducer.BLSPubKey, reducer.BLSSecretKey)
	if err != nil {
		t.Fatal(err)
	}

	message2, err := newSigSetReductionMessage(sigSetHash, winningBlockHash, 1, 2,
		reducer.BLSPubKey, reducer.BLSSecretKey)
	if err != nil {
		t.Fatal(err)
	}

	// send a selection result to trigger the first vote
	eventBus.Publish(msg.SelectionResultTopic, bytes.NewBuffer(sigSetHash))

	// get first vote
	<-outgoingReductionChannel

	// now send the messages and kick off a reduction cycle
	eventBus.Publish(string(topics.SigSetReduction), message1)
	eventBus.Publish(string(topics.SigSetReduction), message2)

	// wait for result
	<-reductionResultChannel

	// we should have a reduction vote in our outgoingreduction channel
	assert.NotEmpty(t, outgoingReductionChannel)

	// we should have a message in our outgoingagreementchannel
	assert.NotEmpty(t, outgoingAgreementChannel)

	// Kill goroutine
	eventBus.Publish(msg.QuitTopic, nil)
}

func TestQueueBlockReduction(t *testing.T) {
	verifyEd25519SignatureFunc := func(*bytes.Buffer) error {
		return nil
	}

	eventBus := wire.New()

	// subscribe to the reductionresult topic
	reductionResultChannel := make(chan *bytes.Buffer, 3)
	eventBus.Subscribe(msg.ReductionResultTopic, reductionResultChannel)

	// subscribe to provisioneradded topic
	provisionerAddedChannel := make(chan *bytes.Buffer, 3)
	eventBus.Subscribe(msg.ProvisionerAddedTopic, provisionerAddedChannel)

	// Create a committee store
	committeeStore := user.NewCommitteeStore(eventBus)
	go committeeStore.Listen()

	// make three bls key pairs
	pubKeyBLS1, privKeyBLS1, err := bls.GenKeyPair(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	pubKeyBLS2, privKeyBLS2, err := bls.GenKeyPair(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	pubKeyBLS3, privKeyBLS3, err := bls.GenKeyPair(rand.Reader)
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

	// make block hash
	blockHash, err := crypto.RandEntropy(32)
	if err != nil {
		t.Fatal(err)
	}

	// make three messages
	message1, err := newBlockReductionMessage(blockHash, 2, 1, pubKeyBLS1, privKeyBLS1)
	if err != nil {
		t.Fatal(err)
	}

	message2, err := newBlockReductionMessage(blockHash, 2, 1, pubKeyBLS2, privKeyBLS2)
	if err != nil {
		t.Fatal(err)
	}

	message3, err := newBlockReductionMessage(blockHash, 2, 1, pubKeyBLS3, privKeyBLS3)
	if err != nil {
		t.Fatal(err)
	}

	// Set up reducer with a short timeout
	timerLength := 100 * time.Millisecond
	keys, err := user.NewRandKeys()
	if err != nil {
		t.Fatal(err)
	}

	reducer := reduction.NewReducer(eventBus, timerLength, verifyEd25519SignatureFunc,
		committeeStore, keys)

	go reducer.Listen()

	// initialise reducer
	roundBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(roundBytes[:8], 1)
	eventBus.Publish(msg.InitializationTopic, bytes.NewBuffer(roundBytes))

	// now send the messages and get them queued
	eventBus.Publish(string(topics.BlockReduction), message1)
	eventBus.Publish(string(topics.BlockReduction), message2)
	eventBus.Publish(string(topics.BlockReduction), message3)

	// create three messages for the next step
	message4, err := newBlockReductionMessage(blockHash, 2, 2, pubKeyBLS1, privKeyBLS1)
	if err != nil {
		t.Fatal(err)
	}

	message5, err := newBlockReductionMessage(blockHash, 2, 2, pubKeyBLS2, privKeyBLS2)
	if err != nil {
		t.Fatal(err)
	}

	message6, err := newBlockReductionMessage(blockHash, 2, 2, pubKeyBLS3, privKeyBLS3)
	if err != nil {
		t.Fatal(err)
	}

	// send the messages and get them queued
	eventBus.Publish(string(topics.BlockReduction), message4)
	eventBus.Publish(string(topics.BlockReduction), message5)
	eventBus.Publish(string(topics.BlockReduction), message6)

	// update the round
	roundUpdateBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(roundUpdateBytes[:8], 2)
	eventBus.Publish(msg.RoundUpdateTopic, bytes.NewBuffer(roundUpdateBytes))

	// check if we got a proper result
	m := <-reductionResultChannel
	assert.Equal(t, blockHash, m.Bytes())

	// Kill goroutine
	eventBus.Publish(msg.QuitTopic, nil)
}

func newProvisionerMessage(pubKeyBLS *bls.PublicKey, amount uint64) (*bytes.Buffer, error) {
	buffer := new(bytes.Buffer)
	pubKeyEd, err := crypto.RandEntropy(32)
	if err != nil {
		return nil, err
	}

	if err := encoding.Write256(buffer, pubKeyEd); err != nil {
		return nil, err
	}

	if err := encoding.WriteVarBytes(buffer, pubKeyBLS.Marshal()); err != nil {
		return nil, err
	}

	if err := encoding.WriteUint64(buffer, binary.LittleEndian, amount); err != nil {
		return nil, err
	}

	return buffer, nil
}

func newBlockReductionMessage(blockHash []byte, round uint64, step uint8,
	pubKeyBLS *bls.PublicKey, privKeyBLS *bls.SecretKey) (*bytes.Buffer, error) {

	buffer := new(bytes.Buffer)
	if err := encoding.Write256(buffer, blockHash); err != nil {
		return nil, err
	}

	signedBlockHash, err := bls.Sign(privKeyBLS, pubKeyBLS, blockHash)
	if err != nil {
		return nil, err
	}

	if err := encoding.WriteBLS(buffer, signedBlockHash.Compress()); err != nil {
		return nil, err
	}

	if err := encoding.WriteVarBytes(buffer, pubKeyBLS.Marshal()); err != nil {
		return nil, err
	}

	if err := encoding.WriteUint64(buffer, binary.LittleEndian, round); err != nil {
		return nil, err
	}

	if err := encoding.WriteUint8(buffer, step); err != nil {
		return nil, err
	}

	return buffer, nil
}

func newSigSetReductionMessage(sigSetHash, winningblockHash []byte, round uint64,
	step uint8, pubKeyBLS *bls.PublicKey, privKeyBLS *bls.SecretKey) (*bytes.Buffer,
	error) {

	buffer := new(bytes.Buffer)
	if err := encoding.Write256(buffer, sigSetHash); err != nil {
		return nil, err
	}

	signedSigSetHash, err := bls.Sign(privKeyBLS, pubKeyBLS, sigSetHash)
	if err != nil {
		return nil, err
	}

	if err := encoding.WriteBLS(buffer, signedSigSetHash.Compress()); err != nil {
		return nil, err
	}

	if err := encoding.WriteVarBytes(buffer, pubKeyBLS.Marshal()); err != nil {
		return nil, err
	}

	if err := encoding.WriteUint64(buffer, binary.LittleEndian, round); err != nil {
		return nil, err
	}

	if err := encoding.WriteUint8(buffer, step); err != nil {
		return nil, err
	}

	if err := encoding.Write256(buffer, winningblockHash); err != nil {
		return nil, err
	}

	return buffer, nil
}
