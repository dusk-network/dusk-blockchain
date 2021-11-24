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
	"errors"
	"math/big"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/keys"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
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

// VerifyStateTransition returns all ContractCalls passed
// height. It returns those ContractCalls deemed valid.
func (p *PermissiveExecutor) VerifyStateTransition(ctx context.Context, cc []ContractCall, height uint64) ([]ContractCall, error) {
	time.Sleep(stateTransitionDelay)
	return cc, nil
}

// FilterTransactions returns all ContractCalls deemed valid.
func (p *PermissiveExecutor) FilterTransactions(ctx context.Context, txs []ContractCall) ([]ContractCall, error) {
	time.Sleep(stateTransitionDelay)
	return txs, nil
}

// ExecuteStateTransition performs a global state mutation and steps the
// block-height up.
func (p *PermissiveExecutor) ExecuteStateTransition(ctx context.Context, cc []ContractCall, height uint64) (user.Provisioners, []byte, error) {
	time.Sleep(stateTransitionDelay)
	return *p.P, make([]byte, 32), nil
}

// GetProvisioners returns current state of provisioners.
func (p *PermissiveExecutor) GetProvisioners(ctx context.Context) (user.Provisioners, error) {
	return *p.P, nil
}

// GetHeight returns current state of rusk height.
func (p *PermissiveExecutor) GetHeight(ctx context.Context) (uint64, error) {
	return 0, nil
}

// MockProxy mocks a proxy for ease of testing.
type MockProxy struct {
	Pr Provider
	V  UnconfirmedTxProber
	KM KeyMaster
	E  Executor
}

// Provider ...
func (m MockProxy) Provider() Provider { return m.Pr }

type mockVerifier struct {
	verifyTransactionLatency time.Duration
}

func (v *mockVerifier) VerifyTransaction(ctx context.Context, cc ContractCall) error {
	if IsMockInvalid(cc) {
		return errors.New("Invalid transaction")
	}

	if v.verifyTransactionLatency > 0 {
		time.Sleep(v.verifyTransactionLatency)
	}

	return nil
}

func (v *mockVerifier) CalculateBalance(ctx context.Context, vkBytes []byte, txs []ContractCall) (uint64, error) {
	return uint64(0), nil
}

// Prober returns a UnconfirmedTxProber that is capable of checking invalid mocked up transactions.
func (m MockProxy) Prober() UnconfirmedTxProber {
	return &mockVerifier{}
}

// ProberWithParams instantiates a mockVerifier with a latency value for VerifyTransaction.
func (m MockProxy) ProberWithParams(verifyTransactionLatency time.Duration) UnconfirmedTxProber {
	return &mockVerifier{verifyTransactionLatency}
}

// KeyMaster ...
func (m MockProxy) KeyMaster() KeyMaster { return m.KM }

// Executor ...
func (m MockProxy) Executor() Executor { return m.E }

// MockKeys mocks the keys.
func MockKeys() (*keys.SecretKey, *keys.PublicKey) {
	sk := keys.NewSecretKey()
	pk := keys.NewPublicKey()

	keys.USecretKey(RuskSecretKey(), sk)
	keys.UPublicKey(RuskPublicKey(), pk)

	return sk, pk
}

/******************/
/** ContractCall **/
/******************/

// RandContractCall returns a random ContractCall.
func RandContractCall() ContractCall {
	switch RandTxType() {
	case Stake:
		return RandStakeTx(0)
	case Bid:
		return RandBidTx(0)
	case Tx:
		return RandTx()
	default:
		return RandTx()
	}
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

	for i := 0; i < invalid; {
		// Pick a random tx within the set and invalidate it until we reach the
		// invalid amount.
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(amount)))
		if err != nil {
			panic(err)
		}

		if !IsMockInvalid(cc[idx.Int64()]) {
			Invalidate(cc[idx.Int64()])
			i++
		}
	}

	if includeCoinbase {
		coinbase := RandDistributeTx(RandUint64(), 30)
		return append([]ContractCall{coinbase}, cc...)
	}

	return cc
}

/************/
/**** TX ****/
/************/

// RandTx returns a random transaction. The randomization includes the amount,
// the fee, the blinding factor and whether the transaction is obfuscated or
// otherwise.
func RandTx() *Transaction {
	bf := make([]byte, 32)
	if _, err := rand.Read(bf); err != nil {
		panic(err)
	}

	return MockTx(RandBool(), bf, true)
}

// MockTx mocks a transfer transaction. For simplicity it includes a single
// output with the amount specified. The blinding factor can be left to nil if
// the test is not interested in Transaction equality/differentiation.
// Otherwise it can be used to identify/differentiate the transaction.
func MockTx(obfuscated bool, blindingFactor []byte, randomized bool) *Transaction {
	ccTx := NewTransaction()
	rtx := mockRuskTx(obfuscated, blindingFactor, randomized)

	if err := UTransaction(rtx, ccTx); err != nil {
		panic(err)
	}

	return ccTx
}

/****************/
/** DISTRIBUTE **/
/****************/

// RandDistributeTx creates a random distribute transaction.
func RandDistributeTx(reward uint64, provisionerNr int) *Transaction {
	rew := reward
	if reward == uint64(0) {
		rew = RandUint64()
	}

	ps := make([][]byte, provisionerNr)
	for i := 0; i < provisionerNr; i++ {
		ps[i] = Rand32Bytes()
	}

	// _, pk := RandKeys()
	tx := RandTx()
	// set the output to 0
	tx.Payload.Notes[0].Commitment = make([]byte, 32)
	buf := new(bytes.Buffer)
	// if err := encoding.WriteVarInt(buf, uint64(provisionerNr)); err != nil {
	// 	panic(err)
	// }

	// for _, pk := range ps {
	// 	if err := encoding.Write256(buf, pk); err != nil {
	// 		panic(err)
	// 	}
	// }

	if err := encoding.WriteUint64LE(buf, rew); err != nil {
		panic(err)
	}

	tx.Payload.CallData = buf.Bytes()
	tx.Payload.Nullifiers = make([][]byte, 0)
	tx.TxType = Distribute

	return tx
}

/************/
/** STAKE **/
/************/

// RandStakeTx creates a random stake transaction. If the expiration
// is <1, then it is randomly set.
func RandStakeTx(expiration uint64) *Transaction {
	if expiration < 1 {
		expiration = RandUint64()
	}

	blsKey := make([]byte, 96)
	if _, err := rand.Read(blsKey); err != nil {
		panic(err)
	}

	return MockStakeTx(expiration, blsKey, true)
}

// MockStakeTx creates a StakeTransaction.
func MockStakeTx(expiration uint64, blsKey []byte, randomized bool) *Transaction {
	stx := NewTransaction()
	rtx := mockRuskTx(false, Rand32Bytes(), randomized)

	if err := UTransaction(rtx, stx); err != nil {
		panic(err)
	}

	buf := new(bytes.Buffer)
	if err := encoding.WriteUint64LE(buf, expiration); err != nil {
		panic(err)
	}

	if err := encoding.WriteVarBytes(buf, blsKey); err != nil {
		panic(err)
	}

	stx.Payload.CallData = buf.Bytes()
	stx.TxType = Stake

	return stx
}

/*********/
/** BID **/
/*********/

// RandBidTx creates a random bid transaction. If the expiration
// is <1, then it is randomly set.
func RandBidTx(expiration uint64) *Transaction {
	if expiration < 1 {
		expiration = RandUint64()
	}

	return MockBidTx(expiration, Rand32Bytes(), Rand32Bytes(), true)
}

// MockBidTx creates a BidTransaction.
func MockBidTx(expiration uint64, edPk, seed []byte, randomized bool) *Transaction {
	stx := NewTransaction()

	// Amount is set directly in the underlying ContractCallTx.
	rtx := mockRuskTx(true, Rand32Bytes(), randomized)
	if err := UTransaction(rtx, stx); err != nil {
		panic(err)
	}

	buf := new(bytes.Buffer)

	// M, Commitment, R
	for i := 0; i < 3; i++ {
		if err := encoding.Write256(buf, Rand32Bytes()); err != nil {
			panic(err)
		}
	}

	if err := encoding.Write256(buf, edPk); err != nil {
		panic(err)
	}

	if err := encoding.Write256(buf, seed); err != nil {
		panic(err)
	}

	if err := encoding.WriteUint64LE(buf, expiration); err != nil {
		panic(err)
	}

	stx.Payload.CallData = buf.Bytes()
	stx.TxType = Bid

	return stx
}

// MockDeterministicBid creates a deterministic bid, where none of the fields
// are subject to randomness. This creates predictability in the output of the
// hash calculation, and is useful for testing purposes.
func MockDeterministicBid(expiration uint64, edPk, seed []byte) *Transaction {
	stx := NewTransaction()

	// amount is set directly in the underlying ContractCallTx
	rtx := mockRuskTx(true, make([]byte, 32), false)
	if err := UTransaction(rtx, stx); err != nil {
		panic(err)
	}

	buf := new(bytes.Buffer)

	// M, Commitment, R
	for i := 0; i < 3; i++ {
		if err := encoding.Write256(buf, make([]byte, 32)); err != nil {
			panic(err)
		}
	}

	if err := encoding.Write256(buf, edPk); err != nil {
		panic(err)
	}

	if err := encoding.Write256(buf, seed); err != nil {
		panic(err)
	}

	if err := encoding.WriteUint64LE(buf, expiration); err != nil {
		panic(err)
	}

	stx.Payload.CallData = buf.Bytes()
	stx.TxType = Bid

	return stx
}

/**************************/
/** Transfer Transaction **/
/**************************/

func mockRuskTx(obfuscated bool, blindingFactor []byte, randomized bool) *rusk.Transaction {
	anchorBytes := make([]byte, 32)
	if randomized {
		anchorBytes = Rand32Bytes()
	}

	nullifierBytes := make([]byte, 32)
	if randomized {
		nullifierBytes = Rand32Bytes()
	}

	if obfuscated {
		pl := &TransactionPayload{
			Anchor:        anchorBytes,
			Nullifiers:    [][]byte{nullifierBytes},
			Notes:         []*Note{mockObfuscatedOutput(blindingFactor)},
			Fee:           MockFee(randomized),
			Crossover:     MockCrossover(randomized),
			SpendingProof: []byte{0xaa, 0xbb},
			CallData:      make([]byte, 0),
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

	pl := &TransactionPayload{
		Anchor:        anchorBytes,
		Nullifiers:    [][]byte{Rand32Bytes()},
		Notes:         []*Note{mockTransparentOutput(blindingFactor)},
		Fee:           MockFee(randomized),
		Crossover:     MockCrossover(randomized),
		SpendingProof: []byte{0xab, 0xbc},
		CallData:      make([]byte, 0),
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

// RuskTx is the mock of a ContractCallTx.
func RuskTx() *rusk.Transaction {
	pl := &TransactionPayload{
		Anchor:        make([]byte, 32),
		Nullifiers:    [][]byte{Rand32Bytes()},
		Notes:         []*Note{mockTransparentOutput(Rand32Bytes())},
		Fee:           MockFee(false),
		Crossover:     MockCrossover(false),
		SpendingProof: []byte{0xaa, 0xbb},
		CallData:      make([]byte, 0),
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

func mockTransparentOutput(blindingFactor []byte) *Note {
	return MockTransparentNote(blindingFactor)
}

// MockTransparentNote is a transparent note.
func MockTransparentNote(blindingFactor []byte) *Note {
	return &Note{
		Nonce:         make([]byte, 32),
		PkR:           make([]byte, 32),
		Commitment:    blindingFactor,
		Randomness:    make([]byte, 32),
		EncryptedData: make([]byte, 96),
	}
}

// MockObfuscatedOutput returns a Note with the amount hashed. To
// allow for equality checking and retrieval, an encrypted blinding factor can
// also be provided.
// Despite the unsofisticated mocking, the hashing should be enough since the
// node has no way to decode obfuscation as this is delegated to RUSK.
func MockObfuscatedOutput(blindingFactor []byte) *Note {
	return &Note{
		Nonce:         make([]byte, 32),
		PkR:           make([]byte, 32),
		Commitment:    blindingFactor,
		Randomness:    make([]byte, 32),
		EncryptedData: make([]byte, 96),
	}
}

func mockObfuscatedOutput(blindingFactor []byte) *Note {
	return MockObfuscatedOutput(blindingFactor)
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
		ValueComm:     valueCommBytes,
		Nonce:         nonceBytes,
		EncryptedData: make([]byte, 96),
	}
}

// MockFee returns a mocked Fee struct.
func MockFee(randomized bool) *Fee {
	rBytes := make([]byte, 32)
	if randomized {
		rBytes = Rand32Bytes()
	}

	pkRBytes := make([]byte, 32)
	if randomized {
		pkRBytes = Rand32Bytes()
	}

	return &Fee{
		GasLimit: 50000,
		GasPrice: 100,
		R:        rBytes,
		PkR:      pkRBytes,
	}
}

/*************/
/** INVALID **/
/*************/

// IsMockInvalid checks whether a ContractCall mock is invalid or not.
func IsMockInvalid(cc ContractCall) bool {
	return bytes.Equal(cc.StandardTx().SpendingProof, []byte("INVALID"))
}

// Invalidate a transaction by marking its Proof field as "INVALID".
func Invalidate(cc ContractCall) {
	cc.StandardTx().SpendingProof = []byte("INVALID")
}

// MockInvalidTx creates an invalid transaction.
func MockInvalidTx() *Transaction {
	tx := NewTransaction()
	input := Rand32Bytes()
	output := mockTransparentOutput(Rand32Bytes())

	// // changing the NoteType to obfuscated with transparent value makes this
	// // transaction invalid
	// output.Note.NoteType = 1

	// fee := MockTransparentOutput(RandUint64(), nil)
	// tx.Payload.Fee = &fee
	tx.Payload.Notes = []*Note{output}
	tx.Payload.Nullifiers = [][]byte{input}
	tx.Payload.SpendingProof = []byte("INVALID")

	return tx
}

/**********/
/** KEYS **/
/**********/

// RandKeys returns a syntactically correct (but semantically rubbish) keypair.
func RandKeys() (keys.SecretKey, keys.PublicKey) {
	sk := keys.SecretKey{
		A: Rand32Bytes(),
		B: Rand32Bytes(),
	}

	pk := keys.PublicKey{
		AG: Rand32Bytes(),
		BG: Rand32Bytes(),
	}

	return sk, pk
}

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
