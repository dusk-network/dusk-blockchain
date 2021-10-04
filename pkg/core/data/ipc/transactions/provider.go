// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package transactions

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/keys"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
)

// TxRequest is a convenient struct to group all parameters needed to create a
// transaction.
type TxRequest struct {
	SK         keys.SecretKey
	PK         keys.PublicKey
	Amount     uint64
	Fee        uint64
	Obfuscated bool
}

// MakeTxRequest populates a TxRequest with all needed data
// ContractCall. As such, it does not need the Public Key.
func MakeTxRequest(sk keys.SecretKey, pk keys.PublicKey, amount uint64, fee uint64, isObfuscated bool) TxRequest {
	return TxRequest{
		SK:         sk,
		PK:         pk,
		Amount:     amount,
		Fee:        fee,
		Obfuscated: isObfuscated,
	}
}

// UnconfirmedTxProber performs verification of contract calls (transactions).
type UnconfirmedTxProber interface {
	// VerifyTransaction verifies a contract call transaction.
	VerifyTransaction(context.Context, ContractCall) error
	// CalculateBalance for transactions on demand. This functionality is used
	// primarily by the mempool which can order RUSK to calculate balance for
	// transactions even if they are unconfirmed.
	CalculateBalance(context.Context, []byte, []ContractCall) (uint64, error)
}

// Provider encapsulates the common Wallet and transaction operations.
type Provider interface {
	// GetBalance returns the balance of the client expressed in (unit?) of
	// DUSK. It accepts the ViewKey as parameter.
	GetBalance(context.Context, keys.ViewKey) (uint64, uint64, error)

	// NewStake creates a staking transaction.
	NewStake(context.Context, []byte, uint64) (*Transaction, error)

	// NewTransaction creates a new transaction using the user's PrivateKey
	// It accepts the PublicKey of the recipient, a value, a fee and whether
	// the transaction should be obfuscated or otherwise.
	NewTransfer(context.Context, uint64, *keys.StealthAddress) (*Transaction, error)
}

// KeyMaster Encapsulates the Key creation and retrieval operations.
type KeyMaster interface {
	// GenerateKeys creates a SecretKey using a []byte as Seed.
	GenerateKeys(context.Context, []byte) (keys.SecretKey, keys.PublicKey, keys.ViewKey, error)

	// TODO: implement
	// FullScanOwnedNotes(ViewKey) (OwnedNotesResponse)
}

// Executor encapsulate the Global State operations.
type Executor interface {
	// VerifyStateTransition takes a bunch of ContractCalls and the block
	// height. It returns those ContractCalls deemed valid.
	VerifyStateTransition(context.Context, []ContractCall, uint64) ([]ContractCall, error)

	// ExecuteStateTransition performs a global state mutation and steps the
	// block-height up.
	ExecuteStateTransition(context.Context, []ContractCall, uint64) (user.Provisioners, error)

	// GetProvisioners returns the current set of provisioners.
	GetProvisioners(ctx context.Context) (user.Provisioners, error)

	// GetHeight returns rusk state height
	GetHeight(ctx context.Context) (uint64, error)
}

// Proxy toward the rusk client.
type Proxy interface {
	Provider() Provider
	Prober() UnconfirmedTxProber
	KeyMaster() KeyMaster
	Executor() Executor
}

type proxy struct {
	stateClient    rusk.StateClient
	keysClient     rusk.KeysClient
	transferClient rusk.TransferClient
	stakeClient    rusk.StakeServiceClient
	walletClient   rusk.WalletClient
	txTimeout      time.Duration
	timeout        time.Duration
}

// NewProxy creates a new Proxy.
func NewProxy(stateClient rusk.StateClient, keysClient rusk.KeysClient, transferClient rusk.TransferClient,
	stakeClient rusk.StakeServiceClient, walletClient rusk.WalletClient, txTimeout, defaultTimeout time.Duration) Proxy {
	return &proxy{
		stateClient:    stateClient,
		keysClient:     keysClient,
		transferClient: transferClient,
		stakeClient:    stakeClient,
		walletClient:   walletClient,
		txTimeout:      txTimeout,
		timeout:        defaultTimeout,
	}
}

// Prober returned by the Proxy.
func (p *proxy) Prober() UnconfirmedTxProber {
	return &verifier{p}
}

// KeyMaster returned by the Proxy.
func (p *proxy) KeyMaster() KeyMaster {
	return &keymaster{p}
}

// Executor returned by the Proxy.
func (p *proxy) Executor() Executor {
	return &executor{p}
}

// Provider returned by the Proxy.
func (p *proxy) Provider() Provider {
	return &provider{p}
}

type verifier struct {
	*proxy
}

// VerifyTransaction verifies a contract call transaction.
func (v *verifier) VerifyTransaction(ctx context.Context, cc ContractCall) error {
	/*
		ccTx, err := EncodeContractCall(cc)
		if err != nil {
			return err
		}

		if res, err := v.client.VerifyTransaction(ctx, ccTx); err != nil {
			return err
		} else if !res.Verified {
			return errors.New("verification failed")
		}

	*/
	return nil
}

// TODO: implement.
func (v *verifier) CalculateBalance(ctx context.Context, b []byte, calls []ContractCall) (uint64, error) {
	return 0, nil
}

type provider struct {
	*proxy
}

// GetBalance returns the balance of the client expressed in (unit?) of
// DUSK. It accepts the ViewKey as parameter.
func (p *provider) GetBalance(ctx context.Context, vk keys.ViewKey) (uint64, uint64, error) {
	req := new(rusk.GetBalanceRequest)
	req.Vk = new(rusk.ViewKey)
	keys.MViewKey(req.Vk, &vk)

	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(p.txTimeout))
	defer cancel()

	res, err := p.walletClient.GetBalance(ctx, req)
	if err != nil {
		return 0, 0, err
	}

	return res.UnlockedBalance, res.LockedBalance, nil
}

// NewStake creates a new transaction using the user's PrivateKey
// It accepts the PublicKey of the recipient, a value, a fee and whether
// the transaction should be obfuscated or otherwise.
func (p *provider) NewStake(ctx context.Context, pubKeyBLS []byte, value uint64) (*Transaction, error) {
	tr := new(rusk.StakeTransactionRequest)
	tr.Value = value
	tr.PublicKeyBls = pubKeyBLS

	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(p.txTimeout))
	defer cancel()

	res, err := p.stakeClient.NewStake(ctx, tr)
	if err != nil {
		return nil, err
	}

	trans := NewTransaction()
	err = UTransaction(res, trans)
	return trans, err
}

// NewTransfer creates a new transaction using the user's PrivateKey
// It accepts the PublicKey of the recipient, a value, a fee and whether
// the transaction should be obfuscated or otherwise.
func (p *provider) NewTransfer(ctx context.Context, value uint64, sa *keys.StealthAddress) (*Transaction, error) {
	tr := new(rusk.TransferTransactionRequest)
	tr.Value = value

	// XXX: In the schema, this is denoted as `bytes`, however, in the
	// `BidTransactionRequest` it is denoted as a `StealthAddress`. This should be
	// homogenized.
	buf := new(bytes.Buffer)
	if err := keys.MarshalStealthAddress(buf, sa); err != nil {
		return nil, err
	}

	tr.Recipient = buf.Bytes()

	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(p.txTimeout))
	defer cancel()

	res, err := p.transferClient.NewTransfer(ctx, tr)
	if err != nil {
		return nil, err
	}

	trans := NewTransaction()
	err = UTransaction(res, trans)
	return trans, err
}

type keymaster struct {
	*proxy
}

// GenerateKeys creates a SecretKey using a []byte as Seed.
func (k *keymaster) GenerateKeys(ctx context.Context, seed []byte) (keys.SecretKey, keys.PublicKey, keys.ViewKey, error) {
	sk := keys.NewSecretKey()
	pk := keys.NewPublicKey()
	vk := keys.NewViewKey()
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
// height. It returns those ContractCalls deemed valid.
func (e *executor) VerifyStateTransition(ctx context.Context, calls []ContractCall, height uint64) ([]ContractCall, error) {
	vstr := new(rusk.VerifyStateTransitionRequest)
	vstr.Txs = make([]*rusk.Transaction, len(calls))

	for i, call := range calls {
		tx := new(rusk.Transaction)
		if err := MTransaction(tx, call.(*Transaction)); err != nil {
			return nil, err
		}

		vstr.Txs[i] = tx
	}

	vstr.Height = height

	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(e.txTimeout))
	defer cancel()

	res, err := e.stateClient.VerifyStateTransition(ctx, vstr)
	if err != nil {
		return nil, err
	}

	for idx := range res.FailedCalls {
		calls[idx], calls[len(calls)-1] = calls[len(calls)-1], calls[idx]
		calls = calls[:len(calls)-1]
	}

	return calls, nil
}

// ExecuteStateTransition performs a global state mutation and steps the
// block-height up.
func (e *executor) ExecuteStateTransition(ctx context.Context, calls []ContractCall, height uint64) (user.Provisioners, error) {
	vstr := new(rusk.ExecuteStateTransitionRequest)
	vstr.Txs = make([]*rusk.Transaction, len(calls))
	vstr.Height = height

	for i, call := range calls {
		tx := new(rusk.Transaction)
		if err := MTransaction(tx, call.(*Transaction)); err != nil {
			return user.Provisioners{}, err
		}

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

func (e *executor) GetProvisioners(ctx context.Context) (user.Provisioners, error) {
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(e.txTimeout))
	defer cancel()

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

func (e *executor) GetHeight(ctx context.Context) (uint64, error) {
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(e.txTimeout))
	defer cancel()

	res, err := e.stateClient.GetHeight(ctx, &rusk.GetHeightRequest{})
	if err != nil {
		return 0, err
	}

	return res.Height, nil
}

// UMember deep copies from the rusk.Provisioner.
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
