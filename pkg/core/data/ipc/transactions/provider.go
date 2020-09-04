package transactions

import (
	"context"
	"errors"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/blindbid"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/keys"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
)

// TxRequest is a convenient struct to group all parameters needed to create a
// transaction
type TxRequest struct {
	SK         keys.SecretKey
	PK         keys.PublicKey
	Amount     uint64
	Fee        uint64
	Obfuscated bool
}

// MakeTxRequest populates a TxRequest with all needed data
// ContractCall. As such, it does not need the Public Key
func MakeTxRequest(sk keys.SecretKey, pk keys.PublicKey, amount uint64, fee uint64, isObfuscated bool) TxRequest {
	return TxRequest{
		SK:         sk,
		PK:         pk,
		Amount:     amount,
		Fee:        fee,
		Obfuscated: isObfuscated,
	}
}

// UnconfirmedTxProber performs verification of contract calls (transactions)
type UnconfirmedTxProber interface {
	// VerifyTransaction verifies a contract call transaction
	VerifyTransaction(context.Context, ContractCall) error
	// CalculateBalance for transactions on demand. This functionality is used
	// primarily by the mempool which can order RUSK to calculate balance for
	// transactions even if they are unconfirmed
	CalculateBalance(context.Context, []byte, []ContractCall) (uint64, error)
}

// Provider encapsulates the common Wallet and transaction operations
type Provider interface {
	// GetBalance returns the balance of the client expressed in (unit?) of
	// DUSK. It accepts the ViewKey as parameter
	GetBalance(context.Context, keys.ViewKey) (uint64, error)

	// NewContractCall creates a non-genesis smart contract related transaction
	NewContractCall(context.Context, []byte, TxRequest) (ContractCall, error)

	// NewTransaction creates a new transaction using the user's PrivateKey
	// It accepts the PublicKey of the recipient, a value, a fee and whether
	// the transaction should be obfuscated or otherwise
	NewTransactionTx(context.Context, TxRequest) (Transaction, error)
}

// KeyMaster Encapsulates the Key creation and retrieval operations
type KeyMaster interface {
	// GenerateKeys creates a SecretKey using a []byte as Seed
	GenerateKeys(context.Context, []byte) (keys.SecretKey, keys.PublicKey, keys.ViewKey, error)

	// TODO: implement
	// FullScanOwnedNotes(ViewKey) (OwnedNotesResponse)
}

// Executor encapsulate the Global State operations
type Executor interface {
	// VerifyStateTransition takes a bunch of ContractCalls and the block
	// height. It returns those ContractCalls deemed valid
	VerifyStateTransition(context.Context, []ContractCall, uint64) ([]ContractCall, error)

	// ExecuteStateTransition performs a global state mutation and steps the
	// block-height up
	ExecuteStateTransition(context.Context, []ContractCall, uint64) (user.Provisioners, error)
}

// Provisioner encapsulates the operations common to a Provisioner during the
// consensus
type Provisioner interface {
	// VerifyScore created by the BlockGenerator
	VerifyScore(context.Context, uint64, uint8, blindbid.VerifyScoreRequest) error
}

// BlockGenerator encapsulates the operations performed by the BlockGenerator
type BlockGenerator interface {
	// GenerateScore to participate in the block generation lottery
	GenerateScore(context.Context, blindbid.GenerateScoreRequest) (blindbid.GenerateScoreResponse, error)
}

// Proxy toward the rusk client
type Proxy interface {
	Provisioner() Provisioner
	Provider() Provider
	Prober() UnconfirmedTxProber
	KeyMaster() KeyMaster
	Executor() Executor
	BlockGenerator() BlockGenerator
}

type proxy struct {
	stateClient    rusk.StateClient
	keysClient     rusk.KeysClient
	blindbidClient rusk.BlindBidServiceClient
	bidClient      rusk.BidServiceClient
	txTimeout      time.Duration
	timeout        time.Duration
}

// NewProxy creates a new Proxy
func NewProxy(stateClient rusk.StateClient, keysClient rusk.KeysClient, blindbidClient rusk.BlindBidServiceClient, bidClient rusk.BidServiceClient, txTimeout, defaultTimeout time.Duration) Proxy {
	return &proxy{
		stateClient:    stateClient,
		keysClient:     keysClient,
		blindbidClient: blindbidClient,
		bidClient:      bidClient,
		txTimeout:      txTimeout,
		timeout:        defaultTimeout,
	}
}

// Prober returned by the Proxy
func (p *proxy) Prober() UnconfirmedTxProber {
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
	/*
		ccTx, err := EncodeContractCall(cc)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithDeadline(ctx, time.Now().Add(v.timeout))
		defer cancel()
		if res, err := v.client.VerifyTransaction(ctx, ccTx); err != nil {
			return err
		} else if !res.Verified {
			return errors.New("verification failed")
		}

	*/
	return nil
}

// TODO: implement
func (v *verifier) CalculateBalance(ctx context.Context, b []byte, calls []ContractCall) (uint64, error) {
	return 0, nil
}

type provider struct {
	*proxy
}

// GetBalance returns the balance of the client expressed in (unit?) of
// DUSK. It accepts the ViewKey as parameter
func (p *provider) GetBalance(ctx context.Context, vk keys.ViewKey) (uint64, error) {
	/*
		rvk := new(rusk.ViewKey)
		keys.MViewKey(rvk, &vk)
		ctx, cancel := context.WithDeadline(ctx, time.Now().Add(p.timeout))
		defer cancel()
		res, err := p.client.GetBalance(ctx, &rusk.GetBalanceRequest{Vk: rvk})
		if err != nil {
			return 0, err
		}

	*/
	return 0, nil
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
	/*
		tr := new(rusk.NewTransactionRequest)
		MTxRequest(tr, tx)

		ctx, cancel := context.WithDeadline(ctx, time.Now().Add(p.txTimeout))
		defer cancel()
		res, err := p.client.NewTransaction(ctx, tr)
		if err != nil {
			return *trans, err
		}

		if err := UTransaction(res, trans); err != nil {
			return *trans, err
		}
	*/
	return *trans, nil
}

type keymaster struct {
	*proxy
}

// GenerateKeys creates a SecretKey using a []byte as Seed
func (k *keymaster) GenerateKeys(ctx context.Context, seed []byte) (keys.SecretKey, keys.PublicKey, keys.ViewKey, error) {
	sk := new(keys.SecretKey)
	pk := new(keys.PublicKey)
	vk := new(keys.ViewKey)
	gskr := new(rusk.GenerateKeysRequest)
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(k.timeout))
	defer cancel()
	res, err := k.keysClient.GenerateKeys(ctx, gskr)
	if err != nil {
		return *sk, *pk, *vk, err
	}

	keys.USecretKey(res.Sk, sk)
	keys.UPublicKey(res.Pk, pk)
	keys.UViewKey(res.Vk, vk)
	return *sk, *pk, *vk, nil
}

type executor struct {
	*proxy
}

// VerifyStateTransition takes a bunch of ContractCalls and the block
// height. It returns those ContractCalls deemed valid
func (e *executor) VerifyStateTransition(ctx context.Context, calls []ContractCall, height uint64) ([]ContractCall, error) {
	vstr := new(rusk.VerifyStateTransitionRequest)
	vstr.Txs = make([]*rusk.Transaction, len(calls))
	var err error
	for i, call := range calls {
		tx := new(rusk.Transaction)
		MTransaction(tx, call.(*Transaction))
		vstr.Txs[i] = tx
	}
	vstr.Height = height

	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(e.txTimeout))
	defer cancel()
	res, err := e.stateClient.VerifyStateTransition(ctx, vstr)
	if err != nil {
		return nil, err
	}

	validTxs := make([]ContractCall, len(res.FailedCalls))
	for i, idx := range res.FailedCalls {
		validTxs[i] = calls[idx]
	}

	return validTxs, nil
}

// ExecuteStateTransition performs a global state mutation and steps the
// block-height up
func (e *executor) ExecuteStateTransition(ctx context.Context, calls []ContractCall, height uint64) (user.Provisioners, error) {
	vstr := new(rusk.ExecuteStateTransitionRequest)
	vstr.Txs = make([]*rusk.Transaction, len(calls))
	vstr.Height = height
	var err error
	for i, call := range calls {
		tx := new(rusk.Transaction)
		MTransaction(tx, call.(*Transaction))
		vstr.Txs[i] = tx
	}

	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(e.txTimeout))
	defer cancel()
	res, err := e.stateClient.ExecuteStateTransition(ctx, vstr)
	if err != nil {
		return user.Provisioners{}, err
	}

	if !res.Success {
		return user.Provisioners{}, errors.New("unsuccessful state transition function execution")
	}

	provisioners := user.NewProvisioners()
	memberMap := make(map[string]*user.Member)
	pres, err := e.stateClient.GetProvisioners(ctx, &rusk.GetProvisionersRequest{})
	if err != nil {
		return user.Provisioners{}, err
	}

	for i := range pres.Provisioners {
		member := new(user.Member)
		UMember(pres.Provisioners[i], member)
		memberMap[string(member.PublicKeyBLS)] = member
		provisioners.Set.Insert(member.PublicKeyBLS)
	}
	provisioners.Members = memberMap

	return *provisioners, nil
}

type provisioner struct {
	*proxy
}

// VerifyScore to participate in the block generation lottery
func (p *provisioner) VerifyScore(ctx context.Context, round uint64, step uint8, score blindbid.VerifyScoreRequest) error {
	gsr := new(rusk.VerifyScoreRequest)
	blindbid.MVerifyScoreRequest(gsr, &score)

	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(p.timeout))
	defer cancel()
	if _, err := p.blindbidClient.VerifyScore(ctx, gsr); err != nil {
		return err
	}
	return nil
}

type blockgenerator struct {
	*proxy
}

// GenerateScore to participate in the block generation lottery
func (b *blockgenerator) GenerateScore(ctx context.Context, s blindbid.GenerateScoreRequest) (blindbid.GenerateScoreResponse, error) {
	gsr := new(rusk.GenerateScoreRequest)
	blindbid.MGenerateScoreRequest(gsr, &s)

	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(b.txTimeout))
	defer cancel()
	score, err := b.blindbidClient.GenerateScore(ctx, gsr)
	if err != nil {
		return blindbid.GenerateScoreResponse{}, err
	}

	g := new(blindbid.GenerateScoreResponse)
	blindbid.UGenerateScoreResponse(score, g)
	return *g, nil
}

// UMember deep copies from the rusk.Provisioner
func UMember(r *rusk.Provisioner, t *user.Member) {
	t.PublicKeyBLS = make([]byte, len(r.PublicKeyBls))
	copy(t.PublicKeyBLS, r.PublicKeyBls)
	t.Stakes = make([]user.Stake, len(r.Stakes))
	for i := range r.Stakes {
		t.Stakes[i] = user.Stake{
			Amount:      r.Stakes[i].Amount,
			StartHeight: r.Stakes[i].StartHeight,
			EndHeight:   r.Stakes[i].EndHeight,
		}
	}
}
