// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package transactions

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"math/big"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
)

const stateTransitionDelay = 1 * time.Second

// PermissiveExecutor implements the transactions.Executor interface. It
// simulates successful Validation and Execution of State transitions
// all Validation and simulates.
type PermissiveExecutor struct {
	height uint64
	P      *user.Provisioners
}

// MockExecutor returns an instance of PermissiveExecutor.
func MockExecutor(height uint64) *PermissiveExecutor {
	return &PermissiveExecutor{
		height: height,
		P:      user.NewProvisioners(),
	}
}

// VerifyStateTransition ...
func (p *PermissiveExecutor) VerifyStateTransition(context.Context, []ContractCall, uint64, uint64) error {
	time.Sleep(stateTransitionDelay)
	return nil
}

// ExecuteStateTransition ...
func (p *PermissiveExecutor) ExecuteStateTransition(ctx context.Context, cc []ContractCall, blockGasLimit uint64, blockHeight uint64) ([]ContractCall, []byte, error) {
	time.Sleep(stateTransitionDelay)

	result := cc
	if len(cc) == 0 {
		result = []ContractCall{RandTx()}
	}

	return result, make([]byte, 32), nil
}

// GetProvisioners ...
func (p *PermissiveExecutor) GetProvisioners(ctx context.Context) (user.Provisioners, error) {
	return *p.P, nil
}

// GetStateRoot ...
func (p *PermissiveExecutor) GetStateRoot(ctx context.Context) ([]byte, error) {
	return make([]byte, 32), nil
}

// Accept ...
func (p *PermissiveExecutor) Accept(context.Context, []ContractCall, []byte, uint64, uint64) ([]ContractCall, user.Provisioners, []byte, error) {
	return nil, *p.P, make([]byte, 32), nil
}

// Finalize ...
func (p *PermissiveExecutor) Finalize(context.Context, []ContractCall, []byte, uint64, uint64) ([]ContractCall, user.Provisioners, []byte, error) {
	return nil, *p.P, make([]byte, 32), nil
}

// Persist ...
func (p *PermissiveExecutor) Persist(context.Context, []byte) error {
	return nil
}

// Revert ...
func (p *PermissiveExecutor) Revert(ctx context.Context) ([]byte, error) {
	return nil, nil
}

// MockProxy mocks a proxy for ease of testing.
type MockProxy struct {
	V UnconfirmedTxProber

	E Executor
}

type mockVerifier struct {
	verifyTransactionLatency time.Duration
}

func (v *mockVerifier) Preverify(ctx context.Context, tx ContractCall) ([]byte, Fee, error) {
	hash, _ := tx.CalculateHash()
	return hash, Fee{GasLimit: 10, GasPrice: 12}, nil
}

// Prober returns a UnconfirmedTxProber that is capable of checking invalid mocked up transactions.
func (m MockProxy) Prober() UnconfirmedTxProber {
	return &mockVerifier{}
}

// ProberWithParams instantiates a mockVerifier with a latency value for VerifyTransaction.
func (m MockProxy) ProberWithParams(verifyTransactionLatency time.Duration) UnconfirmedTxProber {
	return &mockVerifier{verifyTransactionLatency}
}

// Executor ...
func (m MockProxy) Executor() Executor { return m.E }

/******************/
/** ContractCall **/
/******************/

// RandContractCall returns a random ContractCall.
func RandContractCall() ContractCall {
	return RandTx()
}

// RandContractCalls creates random but syntactically valid amount of
// transactions and an "invalid" amount of invalid transactions.
// The invalid transactions are marked as such since they carry "INVALID" in
// the proof. This because we actually have no way to know if a transaction is
// valid or not without RUSK, since syntactically wrong transactions would be
// discarded when Unmarshalling.
// The set is composed of Stake, Bid and normal Transactions.
// If a coinbase is included, an additional Distribute transaction is added
// at the top. A coinbase is never invalid.
func RandContractCalls(amount, invalid int, includeCoinbase bool) []ContractCall {
	if invalid > amount {
		panic("inconsistent number of invalid transactions wrt the total amount")
	}

	cc := make([]ContractCall, amount)
	for i := 0; i < amount; i++ {
		cc[i] = RandContractCall()
	}

	return cc
}

/************/
/**** TX ****/
/************/

// RandTx mocks a transaction.
func RandTx() *Transaction {
	tx := &Transaction{
		Payload: &TransactionPayload{},

		TxType:   Distribute,
		Version:  2,
		FeeValue: Fee{GasLimit: 10, GasPrice: 99},
	}

	b := Rand32Bytes()
	copy(tx.Hash[:], b)

	return tx
}

// MockTx mocks a transaction.
func MockTx() *Transaction {
	tx := &Transaction{
		Payload: &TransactionPayload{
			Data: make([]byte, 100),
		},

		TxType:   1,
		Version:  2,
		FeeValue: Fee{GasLimit: 10, GasPrice: 99},
	}

	return tx
}

// MockTxWithParams mocks a transactions with specified params.
func MockTxWithParams(txtype TxType, gasSpent uint64) ContractCall {
	t := &Transaction{
		TxType:        txtype,
		GasSpentValue: gasSpent,
	}

	copy(t.Hash[:], Rand32Bytes())
	t.Payload = NewTransactionPayload()

	return t
}

// MockDistributeTx MockTx of type Distribute.
func MockDistributeTx() *Transaction {
	tx := &Transaction{
		Payload: &TransactionPayload{
			Data: make([]byte, 100),
		},

		TxType:  Distribute,
		Version: 2,
	}

	return tx
}

/****************/
/** DISTRIBUTE **/
/****************/

/**************************/
/** Transfer Transaction **/
/**************************/

// RuskTx is the mock of a ContractCallTx.
func RuskTx() *rusk.Transaction {
	pl := &TransactionPayload{
		Data: make([]byte, 0),
	}

	buf := new(bytes.Buffer)
	if err := MarshalTransactionPayload(buf, pl); err != nil {
		// There's no way a mocked transaction payload should fail to
		// marshal.
		panic(err)
	}

	return &rusk.Transaction{
		Payload: buf.Bytes(),
	}
}

/**************************/
/** TX OUTPUTS AND NOTES **/
/**************************/

// MockTransparentNote is a transparent note.
func MockTransparentNote(blindingFactor []byte) *Note {
	return NewNote()
}

// MockObfuscatedOutput returns a Note with the amount hashed. To
// allow for equality checking and retrieval, an encrypted blinding factor can
// also be provided.
// Despite the unsofisticated mocking, the hashing should be enough since the
// node has no way to decode obfuscation as this is delegated to RUSK.
func MockObfuscatedOutput(valueCommitment []byte) *Note {
	note := NewNote()
	note.ValueCommitment = valueCommitment
	return note
}

// MockCrossover returns a mocked Crossover struct.
func MockCrossover(randomized bool) *Crossover {
	valueCommBytes := make([]byte, 32)
	if randomized {
		valueCommBytes = Rand32Bytes()
	}

	nonceBytes := make([]byte, 32)
	if randomized {
		nonceBytes = Rand32Bytes()
	}

	return &Crossover{
		ValueCommitment: valueCommBytes,
		Nonce:           nonceBytes,
		EncryptedData:   make([]byte, 96),
	}
}

// MockFee returns a mocked Fee struct.
func MockFee(randomized bool) *Fee {
	return &Fee{
		GasLimit: 50000,
		GasPrice: 100,
	}
}

/*************/
/** INVALID **/
/*************/

// IsMockInvalid checks whether a ContractCall mock is invalid or not.
func IsMockInvalid(cc ContractCall) bool {
	return false
}

// Invalidate a transaction by marking its Proof field as "INVALID".
func Invalidate(cc ContractCall) {
}

// MockInvalidTx creates an invalid transaction.
func MockInvalidTx() *Transaction {
	tx := NewTransaction()

	return tx
}

/**********/
/** KEYS **/
/**********/

// RuskPublicKey mocks rusk pk.
func RuskPublicKey() *rusk.PublicKey {
	return &rusk.PublicKey{
		AG: make([]byte, 32),
		BG: make([]byte, 32),
	}
}

// RuskSecretKey mocks rusk sk.
func RuskSecretKey() *rusk.SecretKey {
	return &rusk.SecretKey{
		A: make([]byte, 32),
		B: make([]byte, 32),
	}
}

/*************/
/** UTILITY **/
/*************/

// RandUint64 returns a random uint64.
func RandUint64() uint64 {
	bint64 := make([]byte, 8)
	if _, err := rand.Read(bint64); err != nil {
		panic(err)
	}

	return binary.LittleEndian.Uint64(bint64)
}

// RandBlind returns a random BlindingFactor (it is just an alias for
// Rand32Bytes).
var RandBlind = Rand32Bytes

// RandBytes returns a random byte slice of the desired size.
func RandBytes(size int) []byte {
	blind := make([]byte, 32)
	if _, err := rand.Read(blind); err != nil {
		panic(err)
	}

	return blind
}

// Rand32Bytes returns random 32 bytes.
func Rand32Bytes() []byte {
	return RandBytes(32)
}

// RandBool returns a random boolean.
func RandBool() bool {
	return RandUint64()&(1<<63) == 0
}

// RandTxType returns a random TxType.
func RandTxType() TxType {
	t, err := rand.Int(rand.Reader, big.NewInt(8))
	if err != nil {
		panic(err)
	}

	return TxType(uint8(t.Uint64()))
}
