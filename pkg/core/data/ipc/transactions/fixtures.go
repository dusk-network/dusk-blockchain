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
	"encoding/hex"
	"math/big"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
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
	p := user.NewProvisioners()
	_ = p.Add(key.NewRandKeys().BLSPubKey, 1, 1, 1, 0)
	return &PermissiveExecutor{
		height: height,
		P:      p,
	}
}

// VerifyStateTransition ...
func (p *PermissiveExecutor) VerifyStateTransition(context.Context, []ContractCall, uint64, uint64, []byte) ([]byte, error) {
	time.Sleep(stateTransitionDelay)
	return make([]byte, 32), nil
}

// ExecuteStateTransition ...
func (p *PermissiveExecutor) ExecuteStateTransition(ctx context.Context, cc []ContractCall, blockGasLimit uint64, blockHeight uint64, generator []byte) ([]ContractCall, []byte, error) {
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
func (p *PermissiveExecutor) Accept(context.Context, []ContractCall, []byte, uint64, uint64, []byte, *user.Provisioners) ([]ContractCall, user.Provisioners, []byte, error) {
	return nil, *p.P, make([]byte, 32), nil
}

// Finalize ...
func (p *PermissiveExecutor) Finalize(context.Context, []ContractCall, []byte, uint64, uint64, []byte, *user.Provisioners) ([]ContractCall, user.Provisioners, []byte, error) {
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
	anchor := hex.EncodeToString(Rand32Bytes())
	nullifiersLength := "0100000000000000"

	// Generate a random blinding factor
	nullifiers := hex.EncodeToString(Rand32Bytes())
	hexPayload := anchor + nullifiersLength + nullifiers + "01000000000000000122ac26240b04b9d8697a7a6d772d41c6af4a9f2a0bc6448c7e6114e8804ac80cc48dcb7e531ccc3b334ae122d4fd40e242e7d8a85fdb82bd4c9e9621a9a60d04a7360f31dc0426451e57b526451a5c3749c131cd38c80f84b235005cef4d1b02b3badf1200e8f833c22807fb59db2aa2ec69b33477ff68a6c5efbd050da26bd0ffffffffffffffff80382f73d39cf357b17072a65d2c17be9f9880324b97101a75c2bfa8b55cb500a74664a6634a404d8fe7c237c0ac343120b7c2a08c07dcb5bfb478b81639c60c33f1f2ac9d3a88bc9502259709df1ec33841644ef18b5bc79a06c95d1d76240f00e40b540200000001000000000000008fcfc11603cb86871527b0ea685928cb15f2328de0a7f078b23ab5da376059060ed3d05d422e85952935b66044045643cc70f423a86064c58e9b6823c902364d01f0f78a0f69e99f97e76dbf3f89f46509d2d1958461585f2086d13f327603bec0a88460350fd1f8771a4d491fba5a47d44febe54727ef8474247d2dfff8c3bd2d87d999c587c5015c02a914c71256bb95bebe01f5d3281330dd1dc3e41583c8007b4c4cc3f5e948e7c18c4473f16c1bff076eb456382ccc0a62cdb76f617d6319992584556a1ae9b82f57984168deedf5479b0656c80a1c996087c82195748b491004000000000000ad59c1241ac7f7044ce29ad2ec83d6427b62cc8e89b109378adaa24ec7d0180aa7f08aceab4ce0256c152c504f9523bcb1d6281a2177f0a16ebad07e514d8307008174fa7bd588318826a4f53fabde76ff2d401cbd696a6ccbefa1d8fc913540888317cb3cf6808de698ed3e8df725357829e6396becfd1f4ca342990bb155a27cbfa65f03dea62af40d4c9935f6a4878adfe01bb5319a685a83e839c71d2b768860fee5262f6f23a358d25ee9cc4f3ab4bc86196b8e9b6451380b7187c7ff098889cf1bceb3e34816748a74e0d929f53b2744aa2aaa5b2160664024eba71335ec4fa8c070ac0eb0160b54b6a65adf05ad18f6f0dddbe0a86bb0917c6c84dee0f68e17403d52b8b8a20587af513716095d3fe3988f809d3b28bd30d14b79e70fa3b78a377ba8776ac990fabfbb04eff96005607a6c001b5babd88ce7c346e96ca8601059549548115b857a949a504e66b5a174281568812de5d979717d681b5d203517d5ddb3278880dade92e3e209b0deefc386d3926a9ffbd9887edcecd2f59513c71d5a403af10281a1f06c99cd48960532bb9c34a22feecb9ae2db84516ac688b70efd8fa0157f58e89c4d404453b834cb55c4412f5578dc361cd5e0b5b5aa26d31ead0beeb02c815691b827809f6496cdc569689da5c1134aee517d8fd8998422a2b16558781f4cec9fd396bdd42ee29d39be4b4e4056df2622d80f1b300f38b49be928a1f28a5c8a074ca73e51bcd601a4a783211eaea28691cc42192008b2926fa0c375161bb18cfa23ade1109971ef2e0ac312960773debdc5fc6b0dd786061d9e04ec63cb91d70cc0056b3cf68f442624f69b434c87a1e8acf7d885c4ab4878d7a9849b9395ed2031a6e7737026f107cdc94ba0d0defdf19a87324d68a81402411475e5d1b115687ae70772c14adf42bbd3a5c680d52896ee3faeb8ebc7218e0b9805116aa2434384ec4d4e96d216859cfb2942a31e16fd703920b1f93fbe69039f2e1bc98757b589cc634e0fc921113f4f3c9d65c126c0b3c105a7139106b9b809b75c9f68393b6a7c4721b692536d95a875c2147f4a8555535ab3dcf00247107ce4d18656c3e79aa534235ccbc969b9b61c5d217230c0c1a60f0f2d179acc9fe1a022f8c2b1858bd4370e646b6077737a0eecceeb2f550f2b6498e90d4cc3d1510f511346ef542bf7a76578fde1dadd5b9d8165faab5046c7f767ca2077dc52b569e71ebfb1ff3351ca61a59aa643e73f720664837fbac1225255b8170285d0966f7ec911d6f92f906e336ede3434575c71aeb2b21dd3224e6855c202a3ae8891b312a48451c6406681272c15bb887b6bd75509bc7098b22a8f2c81b25123ef7d9513e13e9eedf0888e39c761d08732e307f87cb709fb2e91282fe0e68c265bb8aa4d39faa1dbd354f512eb175ff4f9a5a989c1f524c52b417ec4621bf28b1513d901c412de5ec495214301020000000000000000000000000000000000000000000000000000000000000005000000000000007374616b6584c0db848cf67c6beedd8ac09146f0cc8b25b7e6419059bd09cc58917ca38589736275080dc8e52a4fc4e0df2b54f2e383e75c9004d586a64d06a23408a68c54fa47ece43fba6d45002c3b940972191d887a0c1ad8c576cb73d5f3feb0188b049626321a8779ae66bcd9d29c98f72b71c5b020b208a1df7d0575cf08dc06f5087274567a828976379b3ecfe8e32fe088b8f85afe1a3a62f64a703acb21fcd392b36099b91db25f671f20d619795b2d78727ebe0fb50ecc56e3b56bce592cb578a7169403b0ef9d595069c3e98e03b6236bad9afa4647329956590ed6abd04d0013e18f914937ffd9152dc0e300b90176a73c313af138ff6dea2c291bed4d0abefbbc2bfcb37d4ad56a4241135b318bdb3e73c7184358b4982df204be1747a56a99abbf5a021ddfab7930ef50ff728f76847111512132058ac3fa3aab997a9a18d7b52cebaa9b1dcd8bcc1dc3d9e605e9a20d758a39334fde5bb096f2d1084146bf13119a215e25e29eb786d0c711c8b22c45d4041f76fc9cca8ef6fdb1261be89639883a91c2346ab70264c2cd7330efb27efa44278afd6b7bd576f09b95807a8ee9646485f5d92d1a34c2337a290655b47027bdbfe9589b7ca5c0d27dcf70a087d7e102d345fb12211669565dd663c5900265cc1f6bce4b5deefb324f62fc008c71f5ddb59a26f611aa9bad4de1f225ccfe605c1382295af787e867988eb8b0e53d9467612d52a5e76a9c9d338f6cce995b042b4607b3364ee32922a60052912b16be823ddc769e7f14fabf1a6f72426df03725ffce3a4659a74d850bd51400f169e0a4ca4327acde8555cd571ed23ea7b94b9ab115511912185f04d18a0f3b6bc0df06fc06d7b1215d79f8bb40bf441293cd13ec5aaabb656a83de719e9f9da884c0930a3e8f6b4583a20ee196502608373ccb46f412ec71fdcfbfdec8ea6236610dadd40737c13789d083bdf48e126bfe4c109c3307a7d477262f6da0d7067b9988ec53f818e7db6c463f2c7e2b63d8a1d28f0272ffc3afc25420cb0907815b7c0c1bfa315d833a6b5b7d1b723533ecb0c2b5ef43e6e1276fb76716441be7bacca07a556113fd5fbbe7bec6e8ac34ed9a08c707a89f4740e1b0db92fa0307a799faf0dcfedb6c247cb92aaeb43f1841cef365c5f736700ef456271567cd3320c8a6c93018f8ef7f45af7e3aaeb95a61a1e6e95a2e07b04e1fe617c24a3babf9e4db6e06a3ec71335d96529af30e0103912d8d4e6bc7cc0d2e3c615d1b61e37f1b9fdb1fe70c371633a4c18f0f5a372bb0405d6ad38061b2ed2d71d276778a8f6ca370f01a6ef500353229dfc7007191dbda251253d675f312feb993dbb137f2b0637095c5fcbd6776ce2357d3461fc448926656ba3cd1ffd9ee64bca9096572feb4a1ca05ff366f65c99b4bbc7c2245ec7ac2bbe63f58174203efd2db856ffa26a3b0ae5b774b5ff34593e07be75b8cb1c21a0f45bdf5b695b05d7806878ef71fca3cf08b7b29d323aea961cec065a19cb90a33b99880e371c5a6e3f979143bb750b5de539e42e204f62f9ab4b96462c8e44ab4798b183ddd323642590895258767628619d202f89e07a05cd74216857c73031dc8e61544c74d0ce51a7f7f032ce5d85c0e0d938a569e5739724b74dda9eaa411ebec226dc2f99222d2a910567aca1fb4a2ebd7d1357605ef11b62da57f21aca20ff0c878cdb3e45177b7f2a6667bf52df9c4fb1c86bf65fb80e3040000000000000000b3134a0603dbc3be4cd8bcb4925424b88138f561d87a4acc466b93bc0f783e18d05495f0b00c77b1e477f3b625e9ef0e155f4ef89fc1007f8b106b67c6af1dcfeb9404b450aef328a775666518eac1a2156832a16d86712dd261d594270c3d1000000000000000000010a5d4e8000000b8faffff10040000"

	payloadBytes, err := hex.DecodeString(hexPayload)
	if err != nil {
		panic(err)
	}

	payload := &TransactionPayload{Data: payloadBytes}
	tx := &Transaction{
		Payload: payload,
		TxType:  Transfer,
		Version: 2,
	}

	decoded, err := tx.Decode()
	if err != nil {
		panic(err)
	}

	hash, err := decoded.Hash(tx.TxType)
	if err != nil {
		panic(err)
	}

	copy(tx.Hash[:], hash)

	return tx
}

// MockTx mocks a transaction.
func MockTx() *Transaction {
	return RandTx()
}

// EmptyTx creates an zero-ed transaction.
func EmptyTx() *Transaction {
	tx := &Transaction{
		Payload: &TransactionPayload{
			Data: make([]byte, 100),
		},

		TxType:  1,
		Version: 0,
	}

	return tx
}

// MockTxWithParams mocks a transactions with specified params.
func MockTxWithParams(txtype TxType, gasSpent uint64) ContractCall {
	t := RandTx()
	t.TxType = txtype
	t.GasSpentValue = gasSpent
	return t
}

/**************************/
/** Transfer Transaction **/
/**************************/

// RuskTx is the mock of a ContractCallTx.
func RuskTx() *rusk.Transaction {
	pl := RandTx().Payload

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
		Payload: make([]byte, 64),
	}
}

// RuskSecretKey mocks rusk sk.
func RuskSecretKey() *rusk.SecretKey {
	return &rusk.SecretKey{
		Payload: make([]byte, 32),
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
