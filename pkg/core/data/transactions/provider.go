package transactions

import (
	"context"
	"errors"

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

// ScoreRequest is utilized to produce a score
type ScoreRequest struct {
	D      []byte
	K      []byte
	Y      []byte
	YInv   []byte
	Q      []byte
	Z      []byte
	Seed   []byte
	Bids   []byte
	BidPos uint64
}

// Score encapsulates the score values calculated by rusk
type Score struct {
	Proof []byte
	Score []byte
	Z     []byte
	Bids  []byte
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

	// NewTransaction creates a new transaction using the user's PrivateKey
	// It accepts the PublicKey of the recipient, a value, a fee and whether
	// the transaction should be obfuscated or otherwise
	NewTransactionTx(context.Context, TxRequest) (Transaction, error)
	// NewStakeTx transaction accepts the BLSPubKey of the provisioner and the
	// transaction request
	NewStakeTx(context.Context, []byte, TxRequest) (StakeTransaction, error)
	// NewWithdrawStake creates a WithdrawStake transaction
	NewWithdrawStakeTx(context.Context, TxRequest) (WithdrawStakeTransaction, error)

	// NewBidTx creates a new Blind Bid
	NewBidTx(context.Context, TxRequest) (BidTransaction, error)
	// NewWithdrawBidTx creates a Bid withdrawal request
	NewWithdrawBidTx(context.Context, TxRequest) (WithdrawBidTransaction, error)
}

// KeyMaster Encapsulates the Key creation and retrieval operations
type KeyMaster interface {
	// GenerateSecretKey creates a SecretKey using a []byte as Seed
	// TODO: shouldn't this also return the PublicKey and the ViewKey to spare
	// a roundtrip?
	GenerateSecretKey(context.Context, []byte) (SecretKey, error)

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
	ExecuteStateTransition(context.Context, []ContractCall) (uint64, error)
}

// Provisioner encapsulates the operations common to a Provisioner during the
// consensus
type Provisioner interface {
	// NewSlashTx creates a Slash transaction
	NewSlashTx(context.Context, TxRequest) (SlashTransaction, error)

	// NewDistributeTx creates a new Distribute transaction
	NewDistributeTx(context.Context, TxRequest) (DistributeTransaction, error)
	// NewWithdrawFeesTx creates a new WithdrawFees transaction
	NewWithdrawFeesTx(context.Context, TxRequest) (WithdrawFeesTransaction, error)
}

// BlockGenerator encapsulates the operations performed by the BlockGenerator
type BlockGenerator interface {
	// GenerateScore to participate in the block generation lottery
	GenerateScore(context.Context, ScoreRequest) (Score, error)
}

// Consensus is used to keep the consensus in synch with the BidList and the
// Blockgenerator
type Consensus interface {
	// GetConsensusInfo asks rusk for the storage of the Bid and the Stake
	// contracts
	GetConsensusInfo(context.Context, uint64) (BidList, []ProvisionerData, error)
}

// Proxy toward the rusk client
type Proxy struct {
	client rusk.RuskClient
}

// NewProxy creates a new Proxy
func NewProxy() *Proxy {
	// TODO: MEMPOOL initialize rusk client
	return &Proxy{client: nil}
}

// Verifier returned by the Proxy
func (p *Proxy) Verifier() Verifier {
	return &verifier{p}
}

// KeyMaster returned by the Proxy
func (p *Proxy) KeyMaster() KeyMaster {
	return &keymaster{p}
}

// Executor returned by the Proxy
func (p *Proxy) Executor() Executor {
	return &executor{p}
}

// Provisioner returned by the Proxy
func (p *Proxy) Provisioner() Provisioner {
	return &provisioner{p}
}

// Provider returned by the Proxy
func (p *Proxy) Provider() Provider {
	return &provider{p}
}

// BlockGenerator returned by the Proxy
func (p *Proxy) BlockGenerator() BlockGenerator {
	return &blockgenerator{p}
}

// Consensus returned by the Proxy
func (p *Proxy) Consensus() Consensus {
	return &consensus{p}
}

type verifier struct {
	*Proxy
}

// VerifyTransaction verifies a contract call transaction
func (v *verifier) VerifyTransaction(ctx context.Context, cc ContractCall) error {
	ccTx, err := EncodeContractCall(cc)
	if err != nil {
		return err
	}

	// TODO: change signature within dusk-protobuf
	if res, err := v.client.VerifyTransaction(ctx, ccTx); err != nil {
		return err
	} else if !res.Verified {
		return errors.New("verification failed")
	}

	return nil
}

type provider struct {
	*Proxy
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
func (p *provider) NewStakeTx(ctx context.Context, pubKeyBLS []byte, tx TxRequest) (StakeTransaction, error) {
	stakeTx := new(StakeTransaction)
	tr := new(rusk.NewTransactionRequest)
	MTxRequest(tr, tx)

	str := new(rusk.StakeTransactionRequest)
	str.BlsKey = make([]byte, len(pubKeyBLS))
	copy(str.BlsKey, pubKeyBLS)
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
func (p *provider) NewWithdrawStakeTx(ctx context.Context, tx TxRequest) (WithdrawStakeTransaction, error) {
	wsTx := new(WithdrawStakeTransaction)
	tr := new(rusk.NewTransactionRequest)
	MTxRequest(tr, tx)

	res, err := p.client.NewWithdrawStake(ctx, &rusk.WithdrawStakeTransactionRequest{Tx: tr})
	if err != nil {
		return *wsTx, nil
	}

	if err := UWithdrawStake(res, wsTx); err != nil {
		return *wsTx, err
	}
	return *wsTx, nil
}

// NewBidTx creates a new Blind Bid
// TODO: most likely, the Bid Transaction request will need some more
// parameters
func (p *provider) NewBidTx(ctx context.Context, tx TxRequest) (BidTransaction, error) {
	bidTx := new(BidTransaction)
	tr := new(rusk.NewTransactionRequest)
	MTxRequest(tr, tx)

	str := new(rusk.BidTransactionRequest)
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
func (p *provider) NewWithdrawBidTx(ctx context.Context, tx TxRequest) (WithdrawBidTransaction, error) {
	wsTx := new(WithdrawBidTransaction)
	tr := new(rusk.NewTransactionRequest)
	MTxRequest(tr, tx)

	res, err := p.client.NewWithdrawBid(ctx, &rusk.WithdrawBidTransactionRequest{Tx: tr})
	if err != nil {
		return *wsTx, nil
	}

	if err := UWithdrawBid(res, wsTx); err != nil {
		return *wsTx, err
	}
	return *wsTx, nil
}

type keymaster struct {
	*Proxy
}

// GenerateSecretKey creates a SecretKey using a []byte as Seed
// TODO: shouldn't this also return the PublicKey and the ViewKey to spare
// a roundtrip?
func (k *keymaster) GenerateSecretKey(ctx context.Context, seed []byte) (SecretKey, error) {
	sk := new(SecretKey)
	gskr := new(rusk.GenerateSecretKeyRequest)
	gskr.B = seed
	res, err := k.client.GenerateSecretKey(ctx, gskr)
	if err != nil {
		return *sk, err
	}

	USecretKey(res.Sk, sk)
	return *sk, nil
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
	*Proxy
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

	// TODO: filter based on the indexes within the response once
	// #dusk-protobuf#77 is solved
	_, err = e.client.ValidateStateTransition(ctx, vstr)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// ExecuteStateTransition performs a global state mutation and steps the
// block-height up
func (e *executor) ExecuteStateTransition(ctx context.Context, calls []ContractCall) (uint64, error) {
	vstr := new(rusk.ExecuteStateTransitionRequest)
	vstr.Calls = make([]*rusk.ContractCallTx, len(calls))
	var err error
	for i, call := range calls {
		vstr.Calls[i], err = EncodeContractCall(call)
		if err != nil {
			return 0, err
		}
	}

	res, err := e.client.ExecuteStateTransition(ctx, vstr)
	if err != nil {
		return 0, err
	}

	if !res.Success {
		return 0, errors.New("unsuccessful state transition function execution")
	}

	return res.CurrentHeight, nil
}

type provisioner struct {
	*Proxy
}

// NewSlashTx creates a Slash transaction
// TODO: add the necessary parameters
func (p *provisioner) NewSlashTx(ctx context.Context, tx TxRequest) (SlashTransaction, error) {
	slashTx := new(SlashTransaction)
	tr := new(rusk.NewTransactionRequest)
	MTxRequest(tr, tx)

	st := new(rusk.SlashTransactionRequest)
	st.Tx = tr

	res, err := p.client.NewSlash(ctx, st)
	if err != nil {
		return *slashTx, err
	}
	if err := USlash(res, slashTx); err != nil {
		return *slashTx, err
	}
	return *slashTx, nil
}

// NewDistributeTx creates a new Distribute transaction
func (p *provisioner) NewDistributeTx(ctx context.Context, tx TxRequest) (DistributeTransaction, error) {
	dTx := new(DistributeTransaction)
	tr := new(rusk.NewTransactionRequest)
	MTxRequest(tr, tx)

	dt := new(rusk.DistributeTransactionRequest)
	dt.Tx = tr
	res, err := p.client.NewDistribute(ctx, dt)
	if err != nil {
		return *dTx, err
	}

	if err := UDistribute(res, dTx); err != nil {
		return *dTx, err
	}
	return *dTx, nil
}

// NewWithdrawFeesTx creates a new WithdrawFees transaction
// TODO: add missing parameters like BLS PubKey, BLS Signature and message (?)
func (p *provisioner) NewWithdrawFeesTx(ctx context.Context, tx TxRequest) (WithdrawFeesTransaction, error) {
	feeTx := new(WithdrawFeesTransaction)
	tr := new(rusk.NewTransactionRequest)
	MTxRequest(tr, tx)
	wftr := &rusk.WithdrawFeesTransactionRequest{
		Tx: tr,
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

type blockgenerator struct {
	*Proxy
}

// GenerateScore to participate in the block generation lottery
func (b *blockgenerator) GenerateScore(ctx context.Context, s ScoreRequest) (Score, error) {
	gsr := &rusk.GenerateScoreRequest{
		D:      s.D,
		K:      s.K,
		Y:      s.Y,
		YInv:   s.YInv,
		Q:      s.Q,
		Z:      s.Z,
		Seed:   s.Seed,
		Bids:   s.Bids,
		BidPos: s.BidPos,
	}
	score, err := b.client.GenerateScore(ctx, gsr)
	if err != nil {
		return Score{}, err
	}
	return Score{
		Proof: score.Proof,
		Score: score.Score,
		Z:     score.Z,
		Bids:  score.Bids,
	}, nil
}

type consensus struct {
	*Proxy
}

func (c *consensus) GetConsensusInfo(ctx context.Context, height uint64) (BidList, []ProvisionerData, error) {
	gcir := &rusk.GetConsensusInfoRequest{BlockHeight: height}
	res, err := c.client.GetConsensusInfo(ctx, gcir)
	if err != nil {
		return BidList{}, nil, err
	}

	provisioners := make([]ProvisionerData, len(res.Committee))
	for i := range res.Committee {
		UProvisioner(res.Committee[i], &provisioners[i])
	}

	bidList := BidList{}
	UBidList(res.BidList, &bidList)
	return bidList, provisioners, nil

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
