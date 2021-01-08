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

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/blindbid"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/common"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/keys"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
)

// PermissiveExecutor implements the transactions.Executor interface. It
// simulates successful Validation and Execution of State transitions
// all Validation and simulates
type PermissiveExecutor struct {
	height uint64
	P      *user.Provisioners
}

// MockExecutor returns an instance of PermissiveExecutor
func MockExecutor(height uint64) *PermissiveExecutor {
	return &PermissiveExecutor{
		height: height,
		P:      user.NewProvisioners(),
	}
}

// VerifyStateTransition returns all ContractCalls passed
// height. It returns those ContractCalls deemed valid
func (p *PermissiveExecutor) VerifyStateTransition(ctx context.Context, cc []ContractCall, height uint64) ([]ContractCall, error) {
	return cc, nil
}

// ExecuteStateTransition performs a global state mutation and steps the
// block-height up
func (p *PermissiveExecutor) ExecuteStateTransition(ctx context.Context, cc []ContractCall, height uint64) (user.Provisioners, error) {
	return *p.P, nil
}

// GetProvisioners returns current state of provisioners
func (p *PermissiveExecutor) GetProvisioners(ctx context.Context) (user.Provisioners, error) {
	return *p.P, nil
}

// PermissiveProvisioner mocks verification of scores
type PermissiveProvisioner struct {
}

// VerifyScore returns nil all the time
func (p PermissiveProvisioner) VerifyScore(context.Context, uint64, uint8, blindbid.VerifyScoreRequest) error {
	return nil
}

// MockBlockGenerator mocks a blockgenerator
type MockBlockGenerator struct{}

// GenerateScore obeys the BlockGenerator interface
func (b MockBlockGenerator) GenerateScore(context.Context, blindbid.GenerateScoreRequest) (blindbid.GenerateScoreResponse, error) {
	limit, _ := big.NewInt(0).SetString("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", 16)
	prove, _ := crypto.RandEntropy(256)
	prover, _ := crypto.RandEntropy(32)
	score, _ := crypto.RandEntropy(32)
	scoreInt := big.NewInt(0).SetBytes(score)

	// making sure that the score exceeds the threshold
	for scoreInt.Cmp(limit) <= 0 {
		score, _ = crypto.RandEntropy(32)
		scoreInt = big.NewInt(0).SetBytes(score)
	}

	return blindbid.GenerateScoreResponse{
		BlindbidProof:  &common.Proof{Data: prove},
		Score:          &common.BlsScalar{Data: score},
		ProverIdentity: &common.BlsScalar{Data: prover},
	}, nil
}

// MockProxy mocks a proxy for ease of testing
type MockProxy struct {
	P  Provisioner
	Pr Provider
	V  UnconfirmedTxProber
	KM KeyMaster
	E  Executor
	BG BlockGenerator
}

// Provisioner ...
func (m MockProxy) Provisioner() Provisioner { return m.P }

// Provider ...
func (m MockProxy) Provider() Provider { return m.Pr }

type mockVerifier struct{}

func (v *mockVerifier) VerifyTransaction(ctx context.Context, cc ContractCall) error {
	if IsMockInvalid(cc) {
		return errors.New("Invalid transaction")
	}
	return nil
}

func (v *mockVerifier) CalculateBalance(ctx context.Context, vkBytes []byte, txs []ContractCall) (uint64, error) {
	return uint64(0), nil
}

// Prober returns a UnconfirmedTxProber that is capable of checking invalid mocked up transactions
func (m MockProxy) Prober() UnconfirmedTxProber {
	return &mockVerifier{}
}

// KeyMaster ...
func (m MockProxy) KeyMaster() KeyMaster { return m.KM }

// Executor ...
func (m MockProxy) Executor() Executor { return m.E }

// BlockGenerator ...
func (m MockProxy) BlockGenerator() BlockGenerator { return m.BG }

// MockKeys mocks the keys
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

// RandContractCall returns a random ContractCall
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
// If a coinbase is included, // an additional Distribute transaction is added
// at the top. A coinbase is never invalid
func RandContractCalls(amount, invalid int, includeCoinbase bool) []ContractCall {
	if invalid > amount {
		panic("inconsistent number of invalid transactions wrt the total amount")
	}

	cc := make([]ContractCall, amount)
	for i := 0; i < amount; i++ {
		cc[i] = RandContractCall()
	}

	for i := 0; i < invalid; {
		// pick a random tx within the set and invalidate it until we reach the
		// invalid amount
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
// otherwise
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
// Otherwise it can be used to identify/differentiate the transaction
func MockTx(obfuscated bool, blindingFactor []byte, randomized bool) *Transaction {
	ccTx := NewTransaction()
	rtx := mockRuskTx(obfuscated, blindingFactor, randomized)
	UTransaction(rtx, ccTx)

	return ccTx
}

/****************/
/** DISTRIBUTE **/
/****************/

// RandDistributeTx creates a random distribute transaction
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
	tx.TxPayload.Notes[0].Commitment.Data = make([]byte, 32)
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

	tx.TxPayload.CallData = buf.Bytes()
	tx.TxPayload.Nullifiers = make([]*common.BlsScalar, 0)
	tx.TxType = Distribute
	return tx
}

/************/
/** STAKE **/
/************/

// RandStakeTx creates a random stake transaction. If the expiration
// is <1, then it is randomly set
func RandStakeTx(expiration uint64) *Transaction {
	if expiration < 1 {
		expiration = RandUint64()
	}
	return MockStakeTx(expiration, Rand32Bytes(), true)
}

// MockStakeTx creates a StakeTransaction
func MockStakeTx(expiration uint64, blsKey []byte, randomized bool) *Transaction {
	stx := NewTransaction()
	rtx := mockRuskTx(false, Rand32Bytes(), randomized)
	UTransaction(rtx, stx)
	buf := new(bytes.Buffer)
	if err := encoding.WriteVarBytes(buf, blsKey); err != nil {
		panic(err)
	}

	if err := encoding.WriteUint64LE(buf, expiration); err != nil {
		panic(err)
	}

	stx.TxPayload.CallData = buf.Bytes()
	stx.TxType = Stake
	return stx
}

/*********/
/** BID **/
/*********/

// RandBidTx creates a random bid transaction. If the expiration
// is <1, then it is randomly set
func RandBidTx(expiration uint64) *Transaction {
	if expiration < 1 {
		expiration = RandUint64()
	}
	return MockBidTx(expiration, Rand32Bytes(), Rand32Bytes(), true)
}

// MockBidTx creates a BidTransaction
func MockBidTx(expiration uint64, edPk, seed []byte, randomized bool) *Transaction {
	stx := NewTransaction()
	// amount is set directly in the underlying ContractCallTx
	rtx := mockRuskTx(true, Rand32Bytes(), randomized)
	UTransaction(rtx, stx)
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

	stx.TxPayload.CallData = buf.Bytes()
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
	UTransaction(rtx, stx)
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

	stx.TxPayload.CallData = buf.Bytes()
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

	if obfuscated {
		return &rusk.Transaction{
			TxPayload: &rusk.TransactionPayload{
				Anchor: &rusk.BlsScalar{
					Data: anchorBytes,
				},
				Nullifier: []*rusk.BlsScalar{RuskTransparentTxIn()},
				Notes:     []*rusk.Note{mockRuskObfuscatedOutput(blindingFactor)},
				Fee:       MockRuskFee(randomized),
				Crossover: MockRuskCrossover(randomized),
				SpendingProof: &rusk.Proof{
					Data: []byte{0xaa, 0xbb},
				},
				CallData: make([]byte, 0),
			},
		}
	}

	return &rusk.Transaction{
		TxPayload: &rusk.TransactionPayload{
			Anchor: &rusk.BlsScalar{
				Data: anchorBytes,
			},
			Nullifier: []*rusk.BlsScalar{RuskTransparentTxIn()},
			Notes:     []*rusk.Note{mockRuskTransparentOutput(blindingFactor)},
			Fee:       MockRuskFee(randomized),
			Crossover: MockRuskCrossover(randomized),
			SpendingProof: &rusk.Proof{
				Data: []byte{0xab, 0xbc},
			},
			CallData: make([]byte, 0),
		},
	}
}

//RuskTx is the mock of a ContractCallTx
func RuskTx() *rusk.Transaction {
	return &rusk.Transaction{
		TxPayload: &rusk.TransactionPayload{
			Anchor: &rusk.BlsScalar{
				Data: make([]byte, 32),
			},
			Nullifier: []*rusk.BlsScalar{RuskTransparentTxIn()},
			Notes:     []*rusk.Note{RuskTransparentTxOut(Rand32Bytes())},
			Fee:       MockRuskFee(false),
			Crossover: MockRuskCrossover(false),
			SpendingProof: &rusk.Proof{
				Data: []byte{0xaa, 0xbb},
			},
			CallData: make([]byte, 0),
		},
	}
}

/**************************/
/** TX OUTPUTS AND NOTES **/
/**************************/

// MockTransparentOutput returns a transparent Note
func MockTransparentOutput(blindingFactor []byte) Note {
	out := new(Note)
	rto := mockRuskTransparentOutput(blindingFactor)
	UNote(rto, out)
	return *out
}

func mockRuskTransparentOutput(blindingFactor []byte) *rusk.Note {
	return RuskTransparentTxOut(blindingFactor)
}

// RuskTransparentTxOut is a transparent Tx Out Mock
func RuskTransparentTxOut(blindingFactor []byte) *rusk.Note {
	return RuskTransparentNote(blindingFactor)
}

// RuskTransparentNote is a transparent note
func RuskTransparentNote(blindingFactor []byte) *rusk.Note {
	return &rusk.Note{
		Nonce:      &rusk.BlsScalar{Data: make([]byte, 32)},
		PkR:        &rusk.JubJubCompressed{Data: make([]byte, 32)},
		Commitment: &rusk.JubJubCompressed{Data: blindingFactor},
		Randomness: &rusk.JubJubCompressed{Data: make([]byte, 32)},
		// XXX: fix typo in rusk-schema
		EncyptedData: &rusk.PoseidonCipher{
			Data: make([]byte, 96),
		},
	}
}

// MockOfuscatedOutput returns a Note with the amount hashed. To
// allow for equality checking and retrieval, an encrypted blinding factor can
// also be provided.
// Despite the unsofisticated mocking, the hashing should be enough since the
// node has no way to decode obfuscation as this is delegated to RUSK.
func MockOfuscatedOutput(blindingFactor []byte) Note {
	out := &Note{}
	rto := mockRuskObfuscatedOutput(blindingFactor)
	UNote(rto, out)
	return *out
}

func mockRuskObfuscatedOutput(blindingFactor []byte) *rusk.Note {
	return RuskObfuscatedTxOut(blindingFactor)
}

// RuskObfuscatedTxOut is an encrypted Tx Out Mock
func RuskObfuscatedTxOut(blindingFactor []byte) *rusk.Note {
	return &rusk.Note{
		Nonce:      &rusk.BlsScalar{Data: make([]byte, 32)},
		PkR:        &rusk.JubJubCompressed{Data: make([]byte, 32)},
		Commitment: &rusk.JubJubCompressed{Data: blindingFactor},
		Randomness: &rusk.JubJubCompressed{Data: make([]byte, 32)},
		// XXX: fix typo in rusk-schema
		EncyptedData: &rusk.PoseidonCipher{
			Data: make([]byte, 96),
		},
	}
}

// RuskObfuscatedNote is an obfuscated note mock
func RuskObfuscatedNote() *rusk.Note {
	return &rusk.Note{
		Nonce:      &rusk.BlsScalar{Data: make([]byte, 32)},
		PkR:        &rusk.JubJubCompressed{Data: make([]byte, 32)},
		Commitment: &rusk.JubJubCompressed{Data: make([]byte, 32)},
		Randomness: &rusk.JubJubCompressed{Data: make([]byte, 32)},
		// XXX: fix typo in rusk-schema
		EncyptedData: &rusk.PoseidonCipher{
			Data: make([]byte, 96),
		},
	}
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
		ValueComm: &common.JubJubCompressed{
			Data: valueCommBytes,
		},
		Nonce: &common.BlsScalar{
			Data: nonceBytes,
		},
		EncryptedData: &common.PoseidonCipher{
			Data: make([]byte, 96),
		},
	}
}

// MockRuskCrossover returns a mocked rusk.Crossover struct.
func MockRuskCrossover(randomized bool) *rusk.Crossover {
	valueCommBytes := make([]byte, 32)
	if randomized {
		valueCommBytes = Rand32Bytes()
	}

	nonceBytes := make([]byte, 32)
	if randomized {
		nonceBytes = Rand32Bytes()
	}

	return &rusk.Crossover{
		ValueComm: &rusk.JubJubCompressed{
			Data: valueCommBytes,
		},
		Nonce: &rusk.BlsScalar{
			Data: nonceBytes,
		},
		EncyptedData: &rusk.PoseidonCipher{
			Data: make([]byte, 96),
		},
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
		R: &common.JubJubCompressed{
			Data: rBytes,
		},
		PkR: &common.JubJubCompressed{
			Data: pkRBytes,
		},
	}
}

// MockRuskFee returns a mocked rusk.Fee struct.
func MockRuskFee(randomized bool) *rusk.Fee {
	c := new(rusk.Fee)
	MFee(c, MockFee(randomized))
	return c
}

/***************/
/** TX INPUTS **/
/***************/

// RuskTransparentTxIn is a transparent Tx Input mock
func RuskTransparentTxIn() *rusk.BlsScalar {
	return &rusk.BlsScalar{
		Data: make([]byte, 32),
	}
}

// RuskObfuscatedTxIn is an encrypted Tx Input Mock
func RuskObfuscatedTxIn() *rusk.BlsScalar {
	return &rusk.BlsScalar{
		Data: make([]byte, 32),
	}
}

/*************/
/** INVALID **/
/*************/

// IsMockInvalid checks whether a ContractCall mock is invalid or not
func IsMockInvalid(cc ContractCall) bool {
	return bytes.Equal(cc.StandardTx().SpendingProof.Data, []byte("INVALID"))
}

// Invalidate a transaction by marking its Proof field as "INVALID"
func Invalidate(cc ContractCall) {
	cc.StandardTx().SpendingProof.Data = []byte("INVALID")
}

// MockInvalidTx creates an invalid transaction
func MockInvalidTx() *Transaction {
	tx := NewTransaction()

	input := new(common.BlsScalar)
	common.UBlsScalar(RuskTransparentTxIn(), input)

	output := new(Note)
	rout := mockRuskTransparentOutput(Rand32Bytes())
	UNote(rout, output)

	// // changing the NoteType to obfuscated with transparent value makes this
	// // transaction invalid
	// output.Note.NoteType = 1

	// fee := MockTransparentOutput(RandUint64(), nil)
	// tx.TxPayload.Fee = &fee
	tx.TxPayload.Notes = []*Note{output}
	tx.TxPayload.Nullifiers = []*common.BlsScalar{input}
	tx.TxPayload.SpendingProof.Data = []byte("INVALID")
	return tx
}

/**********/
/** KEYS **/
/**********/

// RandKeys returns a syntactically correct (but semantically rubbish) keypair
func RandKeys() (keys.SecretKey, keys.PublicKey) {
	sk := &rusk.SecretKey{
		A: &rusk.JubJubScalar{Data: Rand32Bytes()},
		B: &rusk.JubJubScalar{Data: Rand32Bytes()},
	}
	pk := &rusk.PublicKey{
		AG: &rusk.JubJubCompressed{Data: Rand32Bytes()},
		BG: &rusk.JubJubCompressed{Data: Rand32Bytes()},
	}

	tsk := new(keys.SecretKey)
	keys.USecretKey(sk, tsk)
	tpk := new(keys.PublicKey)
	keys.UPublicKey(pk, tpk)
	return *tsk, *tpk
}

// RuskPublicKey mocks rusk pk
func RuskPublicKey() *rusk.PublicKey {
	return &rusk.PublicKey{
		AG: &rusk.JubJubCompressed{Data: make([]byte, 32)},
		BG: &rusk.JubJubCompressed{Data: make([]byte, 32)},
	}
}

// RuskSecretKey mocks rusk sk
func RuskSecretKey() *rusk.SecretKey {
	return &rusk.SecretKey{
		A: &rusk.JubJubScalar{Data: make([]byte, 32)},
		B: &rusk.JubJubScalar{Data: make([]byte, 32)},
	}
}

/*************/
/** UTILITY **/
/*************/

// RandUint64 returns a random uint64
func RandUint64() uint64 {
	bint64 := make([]byte, 8)
	if _, err := rand.Read(bint64); err != nil {
		panic(err)
	}
	return binary.LittleEndian.Uint64(bint64)
}

// RandBlind returns a random BlindingFactor (it is just an alias for
// Rand32Bytes)
var RandBlind = Rand32Bytes

// RandBytes returns a random byte slice of the desired size
func RandBytes(size int) []byte {
	blind := make([]byte, 32)
	if _, err := rand.Read(blind); err != nil {
		panic(err)
	}
	return blind

}

// Rand32Bytes returns random 32 bytes
func Rand32Bytes() []byte {
	return RandBytes(32)
}

// RandBool returns a random boolean
func RandBool() bool {
	return RandUint64()&(1<<63) == 0
}

// RandTxType returns a random TxType
func RandTxType() TxType {
	t, err := rand.Int(rand.Reader, big.NewInt(8))
	if err != nil {
		panic(err)
	}

	return TxType(uint8(t.Uint64()))
}
