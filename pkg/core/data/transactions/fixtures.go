package transactions

import (
	"context"
	"crypto/rand"
	"encoding/binary"

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

// Verifier ...
func (m MockProxy) Verifier() Verifier { return m.V }

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

/************/
/**** TX ****/
/************/

// MockTx mocks a transfer transaction. For simplicity it includes a single
// output with the amount specified. The blinding factor can be left to nil if
// the test is not interested in Transaction equality/differentiation.
// Otherwise it can be used to identify/differentiate the transaction
func MockTx(amount uint64, fee uint64, obfuscated bool, blindingFactor []byte) (*Transaction, error) {
	ccTx := new(Transaction)
	rtx := mockRuskTx(amount, fee, obfuscated, blindingFactor)
	if err := UTx(rtx, ccTx); err != nil {
		return nil, err
	}

	return ccTx, nil
}

/************/
/** STAKE **/
/************/

// MockStake mocks a stake transaction
func MockStake(amount, expires uint64) (*StakeTransaction, error) {
	ccTx := new(Transaction)
	if err := UTx(RuskTx.ContractCall.(*rusk.ContractCallTx_Tx).Tx, ccTx); err != nil {
		return nil, err
	}
	return &StakeTransaction{
		ContractTx:       &ContractTx{Tx: ccTx},
		BlsKey:           rnd(42),
		Value:            amount,
		ExpirationHeight: expires,
	}, nil
}

/*********/
/** BID **/
/*********/

// MockBid transaction
func MockBid(expires uint64) (*BidTransaction, error) {
	ccTx := new(Transaction)
	if err := UTx(RuskTx.ContractCall.(*rusk.ContractCallTx_Tx).Tx, ccTx); err != nil {
		return nil, err
	}
	return &BidTransaction{
		ContractTx:       &ContractTx{Tx: ccTx},
		M:                rnd(32),
		Commitment:       rnd(32),
		EncryptedValue:   rnd(32),
		EncryptedBlinder: rnd(32),
		Pk:               rnd(32),
		R:                rnd(32),
		Seed:             rnd(32),
		ExpirationHeight: expires,
	}, nil
}

func rnd(size int) []byte {
	b := make([]byte, size)
	_, _ = rand.Read(b)
	return b
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
