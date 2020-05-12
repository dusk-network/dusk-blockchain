package transactions

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	mrand "math/rand"

	"github.com/dusk-network/dusk-crypto/hash"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
)

// PermissiveProvisioner mocks verification of scores
type PermissiveProvisioner struct {
}

// VerifyScore returns nil all the time
func (p PermissiveProvisioner) VerifyScore(context.Context, uint64, uint8, Score) error {
	return nil
}

// NewSlashTx creates a Slash transaction
func (p PermissiveProvisioner) NewSlashTx(context.Context, SlashTxRequest, TxRequest) (SlashTransaction, error) {
	return SlashTransaction{}, nil
}

// NewWithdrawFeesTx creates a new WithdrawFees transaction
func (p PermissiveProvisioner) NewWithdrawFeesTx(context.Context, []byte, []byte, []byte, TxRequest) (WithdrawFeesTransaction, error) {
	return WithdrawFeesTransaction{}, nil
}

// MockProxy mocks a proxy for ease of testing
type MockProxy struct {
	P  Provisioner
	Pr Provider
	V  Verifier
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

// Verifier returns a Verifier that is capable of checking invalid mocked up transactions
func (m MockProxy) Verifier() Verifier {
	return &mockVerifier{}
}

// KeyMaster ...
func (m MockProxy) KeyMaster() KeyMaster { return m.KM }

// Executor ...
func (m MockProxy) Executor() Executor { return m.E }

// BlockGenerator ...
func (m MockProxy) BlockGenerator() BlockGenerator { return m.BG }

// MockKeys mocks the keys
func MockKeys() (*SecretKey, *PublicKey) {
	sk := new(SecretKey)
	pk := new(PublicKey)
	USecretKey(RuskSecretKey, sk)
	UPublicKey(RuskPublicKey, pk)
	return sk, pk
}

/******************/
/** ContractCall **/
/******************/

// RandContractCall returns a random ContractCall
func RandContractCall() ContractCall {
	for {
		switch RandTxType() {
		case Stake:
			return RandStakeTx(0)
		case Bid:
			return RandBidTx(0)
		case Tx:
			return RandTx()
		}
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
		idx := mrand.Intn(amount)
		if !IsMockInvalid(cc[idx]) {
			Invalidate(cc[idx])
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

	return MockTx(RandUint64(), RandUint64(), RandBool(), bf)
}

// MockTx mocks a transfer transaction. For simplicity it includes a single
// output with the amount specified. The blinding factor can be left to nil if
// the test is not interested in Transaction equality/differentiation.
// Otherwise it can be used to identify/differentiate the transaction
func MockTx(amount uint64, fee uint64, obfuscated bool, blindingFactor []byte) *Transaction {
	ccTx := new(Transaction)
	rtx := mockRuskTx(amount, fee, obfuscated, blindingFactor)
	if err := UTx(rtx, ccTx); err != nil {
		panic(err)
	}

	return ccTx
}

/****************/
/** DISTRIBUTE **/
/****************/

// RandDistributeTx creates a random distribute transaction
func RandDistributeTx(reward uint64, provisionerNr int) *DistributeTransaction {
	if reward == uint64(0) {
		reward = RandUint64()
	}

	ps := make([][]byte, provisionerNr)
	for i := 0; i < provisionerNr; i++ {
		ps[i] = Rand32Bytes()
	}

	_, pk := RandKeys()
	return MockDistributeTx(
		RandUint64(),
		ps,
		pk,
	)
}

// MockDistributeTx creates a DistributeTransaction
func MockDistributeTx(reward uint64, provisioners [][]byte, bgPk PublicKey) *DistributeTransaction {
	dtx := newDistribute()
	rtx := mockRuskTx(reward, RandUint64(), false, Rand32Bytes())
	if err := UTx(rtx, dtx.Tx); err != nil {
		panic(err)
	}
	// FIXME: 501 - this field is deprecated
	dtx.TotalReward = reward
	dtx.ProvisionersAddresses = provisioners
	dtx.BgPk = &bgPk
	return dtx
}

/************/
/** STAKE **/
/************/

// RandStakeTx creates a random stake transaction. If the expiration
// is <1, then it is randomly set
func RandStakeTx(expiration uint64) *StakeTransaction {
	if expiration < 1 {
		expiration = RandUint64()
	}
	return MockStakeTx(RandUint64(), expiration, Rand32Bytes())
}

// MockStakeTx creates a StakeTransaction
func MockStakeTx(amount, expiration uint64, blsKey []byte) *StakeTransaction {
	stx := newStake()
	rtx := mockRuskTx(amount, RandUint64(), false, Rand32Bytes())
	if err := UTx(rtx, stx.Tx); err != nil {
		panic(err)
	}
	// FIXME: 501 - this field is deprecated
	stx.Value = amount
	stx.BlsKey = blsKey
	stx.ExpirationHeight = expiration
	return stx
}

/*********/
/** BID **/
/*********/

// RandBidTx creates a random bid transaction. If the expiration
// is <1, then it is randomly set
func RandBidTx(expiration uint64) *BidTransaction {
	if expiration < 1 {
		expiration = RandUint64()
	}
	return MockBidTx(RandUint64(), expiration, Rand32Bytes(), Rand32Bytes())
}

// MockBidTx creates a DistributeTransaction
func MockBidTx(amount, expiration uint64, edPk, seed []byte) *BidTransaction {
	stx := newBid()
	rtx := mockRuskTx(amount, RandUint64(), true, Rand32Bytes())
	if err := UTx(rtx, stx.Tx); err != nil {
		panic(err)
	}
	// FIXME: 501 - EncryptedValue and EncryptedBlinder fields are deprecated
	stx.EncryptedValue = Rand32Bytes()
	stx.EncryptedBlinder = Rand32Bytes()
	stx.Pk = edPk
	stx.R = Rand32Bytes()
	stx.Seed = seed
	stx.ExpirationHeight = expiration
	return stx
}

/**************************/
/** Transfer Transaction **/
/**************************/

func mockRuskTx(amount uint64, fee uint64, obfuscated bool, blindingFactor []byte) *rusk.Transaction {
	feeOut := mockRuskTransparentOutput(fee, nil)
	if obfuscated {
		return &rusk.Transaction{
			Inputs:  []*rusk.TransactionInput{RuskTransparentTxIn},
			Outputs: []*rusk.TransactionOutput{mockRuskObfuscatedOutput(amount, blindingFactor)},
			Fee:     feeOut,
			Proof:   []byte{0xaa, 0xbb},
		}
	}

	return &rusk.Transaction{
		Inputs:  []*rusk.TransactionInput{RuskTransparentTxIn},
		Outputs: []*rusk.TransactionOutput{mockRuskTransparentOutput(amount, blindingFactor)},
		Fee:     feeOut,
		Proof:   []byte{0xab, 0xbc},
	}
}

//RuskTx is the mock of a ContractCallTx
var RuskTx = &rusk.ContractCallTx{
	ContractCall: &rusk.ContractCallTx_Tx{
		Tx: &rusk.Transaction{
			Inputs:  []*rusk.TransactionInput{RuskTransparentTxIn},
			Outputs: []*rusk.TransactionOutput{RuskTransparentTxOut},
			Fee:     RuskTransparentTxOut,
			Proof:   []byte{0xaa, 0xbb},
		},
	},
}

/**************************/
/** TX OUTPUTS AND NOTES **/
/**************************/

// MockTransparentOutput returns a transparent TransactionOutput
func MockTransparentOutput(amount uint64, blindingFactor []byte) TransactionOutput {
	out := new(TransactionOutput)
	rto := mockRuskTransparentOutput(amount, blindingFactor)
	if uerr := UTxOut(rto, out); uerr != nil {
		panic(uerr)
	}
	return *out
}

func mockRuskTransparentOutput(amount uint64, blindingFactor []byte) *rusk.TransactionOutput {
	rto := RuskTransparentTxOut

	rto.Note.Value = &rusk.Note_TransparentValue{
		TransparentValue: amount,
	}

	if blindingFactor != nil {
		rto.Note.BlindingFactor = &rusk.Note_TransparentBlindingFactor{
			TransparentBlindingFactor: &rusk.Scalar{
				Data: blindingFactor,
			},
		}
	}
	return rto
}

// RuskTransparentTxOut is a transparent Tx Out Mock
var RuskTransparentTxOut = &rusk.TransactionOutput{
	Note:           RuskTransparentNote,
	Pk:             RuskPublicKey,
	BlindingFactor: &rusk.Scalar{Data: []byte{0x55, 0x66}},
}

// RuskTransparentNote is a transparent note
var RuskTransparentNote = &rusk.Note{
	NoteType:        0,
	Nonce:           &rusk.Nonce{Bs: []byte{0x11, 0x22}},
	RG:              &rusk.CompressedPoint{Y: []byte{0x33, 0x44}},
	PkR:             &rusk.CompressedPoint{Y: []byte{0x33, 0x44}},
	ValueCommitment: &rusk.Scalar{Data: []byte{0x55, 0x66}},
	BlindingFactor: &rusk.Note_TransparentBlindingFactor{
		TransparentBlindingFactor: &rusk.Scalar{Data: []byte{0x55, 0x66}},
	},
	Value: &rusk.Note_TransparentValue{
		TransparentValue: uint64(122),
	},
}

// MockOfuscatedOutput returns a TransactionOutput with the amount hashed. To
// allow for equality checking and retrieval, an encrypted blinding factor can
// also be provided.
// Despite the unsofisticated mocking, the hashing should be enough since the
// node has no way to decode obfuscation as this is delegated to RUSK.
func MockOfuscatedOutput(amount uint64, blindingFactor []byte) TransactionOutput {
	out := &TransactionOutput{}
	rto := mockRuskObfuscatedOutput(amount, blindingFactor)
	if uerr := UTxOut(rto, out); uerr != nil {
		panic(uerr)
	}
	return *out
}

func mockRuskObfuscatedOutput(amount uint64, blindingFactor []byte) *rusk.TransactionOutput {
	rto := RuskObfuscatedTxOut
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, amount)
	hamount, err := hash.Sha3256(b)
	if err != nil {
		panic(err)
	}

	rto.Note.Value = &rusk.Note_EncryptedValue{
		EncryptedValue: hamount,
	}

	if blindingFactor != nil {
		hfactor, err := hash.Sha3256(blindingFactor)
		if err != nil {
			panic(err)
		}

		rto.Note.BlindingFactor = &rusk.Note_EncryptedBlindingFactor{
			EncryptedBlindingFactor: hfactor,
		}
	}
	return rto
}

// RuskObfuscatedTxOut is an encrypted Tx Out Mock
var RuskObfuscatedTxOut = &rusk.TransactionOutput{
	Note: RuskObfuscatedNote,
	Pk: &rusk.PublicKey{
		AG: &rusk.CompressedPoint{Y: []byte{0x33, 0x44}},
		BG: &rusk.CompressedPoint{Y: []byte{0x33, 0x44}},
	},
	BlindingFactor: &rusk.Scalar{Data: []byte{0x55, 0x66}},
}

// RuskObfuscatedNote is an obfuscated note mock
var RuskObfuscatedNote = &rusk.Note{
	NoteType:        1,
	Nonce:           &rusk.Nonce{},
	RG:              &rusk.CompressedPoint{Y: []byte{0x33, 0x44}},
	PkR:             &rusk.CompressedPoint{Y: []byte{0x33, 0x44}},
	ValueCommitment: &rusk.Scalar{Data: []byte{0x55, 0x66}},
	BlindingFactor: &rusk.Note_EncryptedBlindingFactor{
		EncryptedBlindingFactor: []byte{0x56, 0x67},
	},
	Value: &rusk.Note_EncryptedValue{
		EncryptedValue: []byte{0x12, 0x02},
	},
}

/***************/
/** TX INPUTS **/
/***************/

// RuskTransparentTxIn is a transparent Tx Input mock
var RuskTransparentTxIn = &rusk.TransactionInput{
	Nullifier: &rusk.Nullifier{
		H: &rusk.Scalar{Data: []byte{0x55, 0x66}},
	},
	MerkleRoot: &rusk.Scalar{Data: []byte{0x55, 0x66}},
}

// RuskObfuscatedTxIn is an encrypted Tx Input Mock
var RuskObfuscatedTxIn = &rusk.TransactionInput{
	Nullifier: &rusk.Nullifier{
		H: &rusk.Scalar{Data: []byte{0x55, 0x66}},
	},
	MerkleRoot: &rusk.Scalar{Data: []byte{0x55, 0x66}},
}

/*************/
/** INVALID **/
/*************/

// IsMockInvalid checks whether a ContractCall mock is invalid or not
func IsMockInvalid(cc ContractCall) bool {
	return bytes.Equal(cc.StandardTx().Proof, []byte("INVALID"))
}

// Invalidate a transaction by marking its Proof field as "INVALID"
func Invalidate(cc ContractCall) {
	cc.StandardTx().Proof = []byte("INVALID")
}

// MockInvalidTx creates an invalid transaction
func MockInvalidTx() *Transaction {
	tx := new(Transaction)

	input := new(TransactionInput)
	if err := UTxIn(RuskTransparentTxIn, input); err != nil {
		panic(err)
	}

	output := new(TransactionOutput)
	rout := mockRuskTransparentOutput(RandUint64(), Rand32Bytes())
	if err := UTxOut(rout, output); err != nil {
		panic(err)
	}

	// // changing the NoteType to obfuscated with transparent value makes this
	// // transaction invalid
	// output.Note.NoteType = 1

	fee := MockTransparentOutput(RandUint64(), nil)
	tx.Fee = &fee
	tx.Outputs = []*TransactionOutput{output}
	tx.Inputs = []*TransactionInput{input}
	tx.Proof = []byte("INVALID")
	return tx
}

/******************/
/** INVALID NOTE **/
/******************/

// RuskInvalidNote is an invalid note
var RuskInvalidNote = &rusk.Note{
	NoteType:        1,
	Nonce:           &rusk.Nonce{Bs: []byte{0x11, 0x22}},
	RG:              &rusk.CompressedPoint{Y: []byte{0x33, 0x44}},
	PkR:             &rusk.CompressedPoint{Y: []byte{0x33, 0x44}},
	ValueCommitment: &rusk.Scalar{Data: []byte{0x55, 0x66}},
	BlindingFactor: &rusk.Note_TransparentBlindingFactor{
		TransparentBlindingFactor: &rusk.Scalar{Data: []byte{0x55, 0x66}},
	},
	Value: &rusk.Note_TransparentValue{
		TransparentValue: uint64(122),
	},
}

/**********/
/** KEYS **/
/**********/

// RandKeys returns a syntactically correct (but semantically rubbish) keypair
func RandKeys() (SecretKey, PublicKey) {
	sk := &rusk.SecretKey{
		A: &rusk.Scalar{Data: Rand32Bytes()},
		B: &rusk.Scalar{Data: Rand32Bytes()},
	}
	pk := &rusk.PublicKey{
		AG: &rusk.CompressedPoint{Y: Rand32Bytes()},
		BG: &rusk.CompressedPoint{Y: Rand32Bytes()},
	}

	tsk := new(SecretKey)
	USecretKey(sk, tsk)
	tpk := new(PublicKey)
	UPublicKey(pk, tpk)
	return *tsk, *tpk
}

// RuskPublicKey mocks rusk pk
var RuskPublicKey = &rusk.PublicKey{
	AG: &rusk.CompressedPoint{Y: []byte{0x33, 0x44}},
	BG: &rusk.CompressedPoint{Y: []byte{0x33, 0x44}},
}

// RuskSecretKey mocks rusk sk
var RuskSecretKey = &rusk.SecretKey{
	A: &rusk.Scalar{Data: []byte{0x55, 0x66}},
	B: &rusk.Scalar{Data: []byte{0x55, 0x66}},
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

// Rand32Bytes returns random 32 bytes
func Rand32Bytes() []byte {
	blind := make([]byte, 32)
	if _, err := rand.Read(blind); err != nil {
		panic(err)
	}
	return blind
}

// RandBool returns a random boolean
func RandBool() bool {
	return RandUint64()&(1<<63) == 0
}

// RandTxType returns a random TxType
func RandTxType() TxType {
	t := mrand.Intn(8)
	return TxType(uint8(t))
}
