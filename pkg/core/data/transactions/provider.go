package transactions

import (
	"context"
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
)

// TxRequest is a convenient struct to group all parameters needed to create a
// transaction
type TxRequest struct {
	SK         SecretKey
	PK         PublicKey
	Amount     uint64
	Fee        uint64
	Obfuscated bool
}

// SlashTxRequest is a convenient struct to group all input parameters to
// create a SlashRequest
type SlashTxRequest struct {
	Step      uint8
	Round     uint64
	BlsKey    []byte
	FirstMsg  []byte
	FirstSig  []byte
	SecondMsg []byte
	SecondSig []byte
}

// MakeTxRequest populates a TxRequest with all needed data
// ContractCall. As such, it does not need the Public Key
func MakeTxRequest(sk SecretKey, pk PublicKey, amount uint64, fee uint64, isObfuscated bool) TxRequest {
	return TxRequest{
		SK:         sk,
		PK:         pk,
		Amount:     amount,
		Fee:        fee,
		Obfuscated: isObfuscated,
	}
}

// MakeGenesisTxRequest creates a transaction to be embedded in a genesis
// ContractCall. As such, it does not need the Public Key
func MakeGenesisTxRequest(sk SecretKey, amount uint64, fee uint64, isObfuscated bool) TxRequest {
	return MakeTxRequest(sk, PublicKey{}, amount, fee, isObfuscated)
}

// ScoreRequest is utilized to produce a score
type ScoreRequest struct {
	D     []byte
	K     []byte
	Seed  []byte
	EdPk  []byte
	Round uint64
	Step  uint8
}

// Score encapsulates the score values calculated by rusk
type Score struct {
	Proof    []byte
	Score    []byte
	Seed     []byte
	Identity []byte
}

// Verifier performs verification of contract calls (transactions)
type Verifier interface {
	// VerifyTransaction verifies a contract call transaction
	VerifyTransaction(context.Context, ContractCall) error
}

// Provider encapsulates the common Wallet and transaction operations
type Provider interface {
	// GetBalance returns the balance of the client expressed in (unit?) of
	// DUSK. It accepts the ViewKey as parameter
	GetBalance(context.Context, ViewKey) (uint64, error)

	NewContractCall(context.Context, []byte, TxRequest) (ContractCall, error)

	// NewTransaction creates a new transaction using the user's PrivateKey
	// It accepts the PublicKey of the recipient, a value, a fee and whether
	// the transaction should be obfuscated or otherwise
	NewTransactionTx(context.Context, TxRequest) (Transaction, error)
	// NewStakeTx transaction accepts the BLSPubKey of the provisioner and the
	// transaction request
	NewStakeTx(ctx context.Context, blsKey []byte, expirationHeight uint64, txr TxRequest) (StakeTransaction, error)
	// NewWithdrawStake creates a WithdrawStake transaction
	NewWithdrawStakeTx(context.Context, []byte, []byte, TxRequest) (WithdrawStakeTransaction, error)

	// NewBidTx creates a new Blind Bid. The call accepts a context, the secret
	// `K`, the Edward Public Key `EdPk`, the `seed` and the `expirationHeight`
	NewBidTx(context.Context, []byte, []byte, []byte, uint64, TxRequest) (BidTransaction, error)
	// NewWithdrawBidTx creates a Bid withdrawal request
	NewWithdrawBidTx(context.Context, []byte, []byte, TxRequest) (WithdrawBidTransaction, error)
}

// KeyMaster Encapsulates the Key creation and retrieval operations
type KeyMaster interface {
	// GenerateSecretKey creates a SecretKey using a []byte as Seed
	GenerateSecretKey(context.Context, []byte) (SecretKey, PublicKey, ViewKey, error)

	// Keys returns the PublicKey and ViewKey from the SecretKey
	Keys(context.Context, SecretKey) (PublicKey, ViewKey, error)

	// TODO: implement
	// FullScanOwnedNotes(ViewKey) (OwnedNotesResponse)
}

// Executor encapsulate the Global State operations
type Executor interface {
	// ValidateStateTransition takes a bunch of ContractCalls and the block
	// height. It returns those ContractCalls deemed valid
	ValidateStateTransition(context.Context, []ContractCall, uint64) ([]ContractCall, error)

	// ExecuteStateTransition performs a global state mutation and steps the
	// block-height up
	ExecuteStateTransition(context.Context, []ContractCall) (uint64, user.Provisioners, error)
}

// Provisioner encapsulates the operations common to a Provisioner during the
// consensus
type Provisioner interface {
	// NewSlashTx creates a Slash transaction
	NewSlashTx(context.Context, SlashTxRequest, TxRequest) (SlashTransaction, error)

	// NewWithdrawFeesTx creates a new WithdrawFees transaction using the BLS
	// Key, signature and a Msg
	NewWithdrawFeesTx(context.Context, []byte, []byte, []byte, TxRequest) (WithdrawFeesTransaction, error)

	// VerifyScore created by the BlockGenerator
	VerifyScore(context.Context, uint64, uint8, Score) error
}

// BlockGenerator encapsulates the operations performed by the BlockGenerator
type BlockGenerator interface {
	// GenerateScore to participate in the block generation lottery
	GenerateScore(context.Context, ScoreRequest) (Score, error)

	// NewDistributeTx creates a new Distribute transaction
	NewDistributeTx(ctx context.Context, reward uint64, provisionerAddresses [][]byte, tx TxRequest) (DistributeTransaction, error)
}

// Proxy toward the rusk client
type Proxy interface {
	Provisioner() Provisioner
	Provider() Provider
	Verifier() Verifier
	KeyMaster() KeyMaster
	Executor() Executor
	BlockGenerator() BlockGenerator
}

type proxy struct {
	client rusk.RuskClient
}

// NewProxy creates a new Proxy
func NewProxy(client rusk.RuskClient) Proxy {
	return &proxy{client: client}
}

// Verifier returned by the Proxy
func (p *proxy) Verifier() Verifier {
	return &verifier{p}
}

// KeyMaster returned by the Proxy
func (p *proxy) KeyMaster() KeyMaster {
	return &keymaster{p}
}

// Executor returned by the Proxy
func (p *proxy) Executor() Executor {
	return &executor{p}
}

// Provisioner returned by the Proxy
func (p *proxy) Provisioner() Provisioner {
	return &provisioner{p}
}

// Provider returned by the Proxy
func (p *proxy) Provider() Provider {
	return &provider{p}
}

// BlockGenerator returned by the Proxy
func (p *proxy) BlockGenerator() BlockGenerator {
	return &blockgenerator{p}
}

type verifier struct {
	*proxy
}

// VerifyTransaction verifies a contract call transaction
func (v *verifier) VerifyTransaction(ctx context.Context, cc ContractCall) error {
	ccTx, err := EncodeContractCall(cc)
	if err != nil {
		return err
	}

	if res, err := v.client.VerifyTransaction(ctx, ccTx); err != nil {
		return err
	} else if !res.Verified {
		return errors.New("verification failed")
	}

	return nil
}

type provider struct {
	*proxy
}

// GetBalance returns the balance of the client expressed in (unit?) of
// DUSK. It accepts the ViewKey as parameter
func (p *provider) GetBalance(ctx context.Context, vk ViewKey) (uint64, error) {
	rvk := new(rusk.ViewKey)
	MViewKey(rvk, &vk)
	res, err := p.client.GetBalance(ctx, &rusk.GetBalanceRequest{Vk: rvk})
	if err != nil {
		return 0, err
	}

	return res.Balance, nil
}

// NewContractCall creates a new transaction using the user's PrivateKey
// It accepts the PublicKey of the recipient, a value, a fee and whether
// the transaction should be obfuscated or otherwise
func (p *provider) NewContractCall(ctx context.Context, b []byte, tx TxRequest) (ContractCall, error) {
	//trans := new(Transaction)
	//tr := new(rusk.NewTransactionRequest)
	//MTxRequest(tr, tx)
	//res, err := p.client.NewContractCall(ctx, tr)
	return nil, errors.New("not implemented yet")
}

// NewTransaction creates a new transaction using the user's PrivateKey
// It accepts the PublicKey of the recipient, a value, a fee and whether
// the transaction should be obfuscated or otherwise
func (p *provider) NewTransactionTx(ctx context.Context, tx TxRequest) (Transaction, error) {
	trans := new(Transaction)
	tr := new(rusk.NewTransactionRequest)
	MTxRequest(tr, tx)

	res, err := p.client.NewTransaction(ctx, tr)
	if err != nil {
		return *trans, err
	}

	if err := UTx(res, trans); err != nil {
		return *trans, err
	}
	return *trans, nil
}

// NewStakeTx transaction
func (p *provider) NewStakeTx(ctx context.Context, pubKeyBLS []byte, expirationHeight uint64, tx TxRequest) (StakeTransaction, error) {
	stakeTx := new(StakeTransaction)
	tr := new(rusk.NewTransactionRequest)
	MTxRequest(tr, tx)

	str := new(rusk.StakeTransactionRequest)
	str.BlsKey = make([]byte, len(pubKeyBLS))
	copy(str.BlsKey, pubKeyBLS)
	str.ExpirationHeight = expirationHeight
	res, err := p.client.NewStake(ctx, str)
	if err != nil {
		return *stakeTx, err
	}

	if err := UStake(res, stakeTx); err != nil {
		return *stakeTx, err
	}

	return *stakeTx, nil
}

// NewWithdrawStake creates a WithdrawStake transaction
// TODO: most likely we will need a BLS signature and a BLS public key as
// parameters
func (p *provider) NewWithdrawStakeTx(ctx context.Context, blsKey, sig []byte, tx TxRequest) (WithdrawStakeTransaction, error) {
	wsTx := new(WithdrawStakeTransaction)
	tr := new(rusk.NewTransactionRequest)
	MTxRequest(tr, tx)

	res, err := p.client.NewWithdrawStake(ctx, &rusk.WithdrawStakeTransactionRequest{
		Tx:     tr,
		BlsKey: blsKey,
		Sig:    sig,
	})
	if err != nil {
		return *wsTx, nil
	}

	if err := UWithdrawStake(res, wsTx); err != nil {
		return *wsTx, err
	}
	return *wsTx, nil
}

// NewBidTx creates a new Blind Bid. It uses the secret K, the Edward PK, the
// seed and the block height for unlocking
func (p *provider) NewBidTx(ctx context.Context, k, edPk, seed []byte, expirationHeight uint64, tx TxRequest) (BidTransaction, error) {
	bidTx := new(BidTransaction)
	tr := new(rusk.NewTransactionRequest)
	MTxRequest(tr, tx)

	str := new(rusk.BidTransactionRequest)
	str.Tx = tr
	str.K = k
	str.EdPk = edPk
	str.Seed = seed
	str.ExpirationHeight = expirationHeight

	res, err := p.client.NewBid(ctx, str)
	if err != nil {
		return *bidTx, err
	}

	if err := UBid(res, bidTx); err != nil {
		return *bidTx, err
	}

	return *bidTx, nil
}

// NewWithdrawBidTx creates a Bid withdrawal request
func (p *provider) NewWithdrawBidTx(ctx context.Context, edPk []byte, sig []byte, tx TxRequest) (WithdrawBidTransaction, error) {
	wsTx := new(WithdrawBidTransaction)
	tr := new(rusk.NewTransactionRequest)
	MTxRequest(tr, tx)

	res, err := p.client.NewWithdrawBid(ctx, &rusk.WithdrawBidTransactionRequest{
		Tx:   tr,
		EdPk: edPk,
		Sig:  sig,
	})
	if err != nil {
		return *wsTx, nil
	}

	if err := UWithdrawBid(res, wsTx); err != nil {
		return *wsTx, err
	}
	return *wsTx, nil
}

type keymaster struct {
	*proxy
}

// GenerateSecretKey creates a SecretKey using a []byte as Seed
// TODO: shouldn't this also return the PublicKey and the ViewKey to spare
// a roundtrip?
func (k *keymaster) GenerateSecretKey(ctx context.Context, seed []byte) (SecretKey, PublicKey, ViewKey, error) {
	sk := new(SecretKey)
	pk := new(PublicKey)
	vk := new(ViewKey)
	gskr := new(rusk.GenerateSecretKeyRequest)
	gskr.B = seed
	res, err := k.client.GenerateSecretKey(ctx, gskr)
	if err != nil {
		return *sk, *pk, *vk, err
	}

	USecretKey(res.Sk, sk)
	UPublicKey(res.Pk, pk)
	UViewKey(res.Vk, vk)
	return *sk, *pk, *vk, nil
}

// Keys returns the PublicKey and ViewKey from the SecretKey
func (k *keymaster) Keys(ctx context.Context, sk SecretKey) (PublicKey, ViewKey, error) {
	ruskSk := new(rusk.SecretKey)
	pk, vk := new(PublicKey), new(ViewKey)
	MSecretKey(ruskSk, &sk)
	res, err := k.client.Keys(ctx, ruskSk)
	if err != nil {
		return PublicKey{}, ViewKey{}, err
	}

	UPublicKey(res.Pk, pk)
	UViewKey(res.Vk, vk)
	return *pk, *vk, nil
}

// TODO: implement
// FullScanOwnedNotes(ViewKey) (OwnedNotesResponse)
type executor struct {
	*proxy
}

// ValidateStateTransition takes a bunch of ContractCalls and the block
// height. It returns those ContractCalls deemed valid
func (e *executor) ValidateStateTransition(ctx context.Context, calls []ContractCall, height uint64) ([]ContractCall, error) {
	vstr := new(rusk.ValidateStateTransitionRequest)
	vstr.Calls = make([]*rusk.ContractCallTx, len(calls))
	var err error
	for i, call := range calls {
		vstr.Calls[i], err = EncodeContractCall(call)
		if err != nil {
			return nil, err
		}
	}
	vstr.CurrentHeight = height

	res, err := e.client.ValidateStateTransition(ctx, vstr)
	if err != nil {
		return nil, err
	}

	validTxs := make([]ContractCall, len(res.SuccessfulCalls))
	for i, idx := range res.SuccessfulCalls {
		validTxs[i] = calls[idx]
	}

	return validTxs, nil
}

// ExecuteStateTransition performs a global state mutation and steps the
// block-height up
func (e *executor) ExecuteStateTransition(ctx context.Context, calls []ContractCall) (uint64, user.Provisioners, error) {
	vstr := new(rusk.ExecuteStateTransitionRequest)
	vstr.Calls = make([]*rusk.ContractCallTx, len(calls))
	var err error
	for i, call := range calls {
		vstr.Calls[i], err = EncodeContractCall(call)
		if err != nil {
			return 0, user.Provisioners{}, err
		}
	}

	res, err := e.client.ExecuteStateTransition(ctx, vstr)
	if err != nil {
		return 0, user.Provisioners{}, err
	}

	if !res.Success {
		return 0, user.Provisioners{}, errors.New("unsuccessful state transition function execution")
	}

	provisioners := user.NewProvisioners()
	memberMap := make(map[string]*user.Member)
	for i := range res.Committee {
		member := new(user.Member)
		UMember(res.Committee[i], member)
		memberMap[string(member.PublicKeyBLS)] = member
		provisioners.Set.Insert(member.PublicKeyBLS)
	}
	provisioners.Members = memberMap

	return res.CurrentHeight, *provisioners, nil
}

type provisioner struct {
	*proxy
}

// NewSlashTx creates a Slash transaction
func (p *provisioner) NewSlashTx(ctx context.Context, str SlashTxRequest, tx TxRequest) (SlashTransaction, error) {
	slashTx := new(SlashTransaction)
	tr := new(rusk.NewTransactionRequest)
	MTxRequest(tr, tx)

	st := new(rusk.SlashTransactionRequest)
	st.Tx = tr
	st.Round = str.Round
	st.Step = uint32(str.Step)
	st.BlsKey = str.BlsKey
	st.FirstMsg = str.FirstMsg
	st.FirstSig = str.FirstSig
	st.SecondMsg = str.SecondMsg
	st.SecondSig = str.SecondSig

	res, err := p.client.NewSlash(ctx, st)
	if err != nil {
		return *slashTx, err
	}
	if err := USlash(res, slashTx); err != nil {
		return *slashTx, err
	}
	return *slashTx, nil
}

// NewWithdrawFeesTx creates a new WithdrawFees transaction
// TODO: add missing parameters like BLS PubKey, BLS Signature and message (?)
func (p *provisioner) NewWithdrawFeesTx(ctx context.Context, blsKey, sig, msg []byte, tx TxRequest) (WithdrawFeesTransaction, error) {
	feeTx := new(WithdrawFeesTransaction)
	tr := new(rusk.NewTransactionRequest)
	MTxRequest(tr, tx)
	wftr := &rusk.WithdrawFeesTransactionRequest{
		Tx:     tr,
		BlsKey: blsKey,
		Sig:    sig,
		Msg:    msg,
	}

	wf, err := p.client.NewWithdrawFees(ctx, wftr)
	if err != nil {
		return *feeTx, err
	}

	if err := UWithdrawFees(wf, feeTx); err != nil {
		return *feeTx, err
	}

	return *feeTx, nil
}

// VerifyScore to participate in the block generation lottery
func (p *provisioner) VerifyScore(ctx context.Context, round uint64, step uint8, score Score) error {
	gsr := &rusk.VerifyScoreRequest{
		Proof:    score.Proof,
		Score:    score.Score,
		Identity: score.Identity,
		Seed:     score.Seed,
		Round:    round,
		Step:     uint32(step),
	}
	if _, err := p.client.VerifyScore(ctx, gsr); err != nil {
		return err
	}
	return nil
}

type blockgenerator struct {
	*proxy
}

// GenerateScore to participate in the block generation lottery
func (b *blockgenerator) GenerateScore(ctx context.Context, s ScoreRequest) (Score, error) {
	gsr := &rusk.GenerateScoreRequest{
		D:     s.D,
		K:     s.K,
		Seed:  s.Seed,
		EdPk:  s.EdPk,
		Round: s.Round,
		Step:  uint32(s.Step),
	}
	score, err := b.client.GenerateScore(ctx, gsr)
	if err != nil {
		return Score{}, err
	}
	return Score{
		Proof:    score.Proof,
		Score:    score.Score,
		Seed:     score.Seed,
		Identity: score.Identity,
	}, nil
}

// NewDistributeTx creates a new Distribute transaction
func (b *blockgenerator) NewDistributeTx(ctx context.Context, reward uint64, provisionerAddresses [][]byte, tx TxRequest) (DistributeTransaction, error) {
	dTx := new(DistributeTransaction)
	tr := new(rusk.NewTransactionRequest)
	MTxRequest(tr, tx)

	dt := new(rusk.DistributeTransactionRequest)
	dt.Tx = tr
	dt.TotalReward = reward
	dt.ProvisionersAddresses = make([][]byte, len(provisionerAddresses))
	for i, p := range provisionerAddresses {
		dt.ProvisionersAddresses[i] = make([]byte, len(p))
		copy(dt.ProvisionersAddresses[i], p)
	}
	res, err := b.client.NewDistribute(ctx, dt)
	if err != nil {
		return *dTx, err
	}

	if err := UDistribute(res, dTx); err != nil {
		return *dTx, err
	}
	return *dTx, nil
}

// UMember deep copies from the rusk.Provisioner
func UMember(r *rusk.Provisioner, t *user.Member) {
	t.PublicKeyBLS = make([]byte, len(r.BlsKey))
	copy(t.PublicKeyBLS, r.BlsKey)
	t.Stakes = make([]user.Stake, len(r.Stakes))
	for i := range r.Stakes {
		t.Stakes[i] = user.Stake{
			Amount:      r.Stakes[i].Amount,
			StartHeight: r.Stakes[i].StartHeight,
			EndHeight:   r.Stakes[i].EndHeight,
		}
	}
}

// MTxRequest serializes a TxRequest into its rusk equivalent
func MTxRequest(r *rusk.NewTransactionRequest, t TxRequest) {
	r.Sk = new(rusk.SecretKey)
	MSecretKey(r.Sk, &t.SK)

	r.Recipient = new(rusk.PublicKey)
	MPublicKey(r.Recipient, &t.PK)

	r.Value = t.Amount
	r.Fee = t.Fee
	r.Obfuscated = t.Obfuscated
}
