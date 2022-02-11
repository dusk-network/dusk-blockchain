// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package transactions

import (
	"context"
	"errors"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
)

// UnconfirmedTxProber performs verification of contract calls (transactions).
type UnconfirmedTxProber interface {
	// VerifyTransaction verifies a contract call transaction.
	Preverify(context.Context, ContractCall) ([]byte, Fee, error)
}

// Executor encapsulate the Global State operations.
type Executor interface {
	// VerifyStateTransition performs dry-run state transition to ensure all txs are valid.
	VerifyStateTransition(context.Context, []ContractCall, uint64, uint64) error

	// ExecuteStateTransition performs dry-run state transition to return valid-only set of txs and state hash.
	ExecuteStateTransition(context.Context, []ContractCall, uint64, uint64) ([]ContractCall, []byte, error)

	// Accept creates an ephemeral state transition.
	Accept(context.Context, []ContractCall, []byte, uint64, uint64) (user.Provisioners, []byte, error)

	// Finalize creates a finalized state transition.
	Finalize(context.Context, []ContractCall, []byte, uint64, uint64) (user.Provisioners, []byte, error)

	// GetProvisioners returns the current set of provisioners.
	GetProvisioners(ctx context.Context) (user.Provisioners, error)

	// GetStateRoot returns root hash of the finalized state.
	GetStateRoot(ctx context.Context) ([]byte, error)
}

// Proxy toward the rusk client.
type Proxy interface {
	Prober() UnconfirmedTxProber
	Executor() Executor
}

type proxy struct {
	stateClient rusk.StateClient
	txTimeout   time.Duration
	timeout     time.Duration
}

// NewProxy creates a new Proxy.
func NewProxy(stateClient rusk.StateClient, txTimeout, defaultTimeout time.Duration) Proxy {
	return &proxy{
		stateClient: stateClient,
		txTimeout:   txTimeout,
		timeout:     defaultTimeout,
	}
}

// Prober returned by the Proxy.
func (p *proxy) Prober() UnconfirmedTxProber {
	return &verifier{p}
}

// Executor returned by the Proxy.
func (p *proxy) Executor() Executor {
	return &executor{p}
}

type verifier struct {
	*proxy
}

// Preverify verifies a contract call transaction.
func (v *verifier) Preverify(ctx context.Context, call ContractCall) ([]byte, Fee, error) {
	vstr := new(rusk.PreverifyRequest)

	tx := new(rusk.Transaction)
	if err := MTransaction(tx, call.(*Transaction)); err != nil {
		return nil, Fee{}, err
	}

	vstr.Tx = tx

	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(v.txTimeout))
	defer cancel()

	res, err := v.stateClient.Preverify(ctx, vstr)
	if err != nil {
		return nil, Fee{}, err
	}

	return res.TxHash,
		Fee{GasLimit: res.Fee.GasLimit, GasPrice: res.Fee.GasPrice},
		nil
}

type executor struct {
	*proxy
}

// VerifyStateTransition see also Executor.VerifyStateTransition.
func (e *executor) VerifyStateTransition(ctx context.Context, calls []ContractCall, blockGasLimit, blockHeight uint64) error {
	vstr := new(rusk.VerifyStateTransitionRequest)
	vstr.Txs = make([]*rusk.Transaction, len(calls))

	for i, call := range calls {
		tx := new(rusk.Transaction)
		if err := MTransaction(tx, call.(*Transaction)); err != nil {
			return err
		}

		vstr.Txs[i] = tx
	}

	vstr.BlockHeight = blockHeight
	vstr.BlockGasLimit = blockGasLimit

	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(e.txTimeout))
	defer cancel()

	_, err := e.stateClient.VerifyStateTransition(ctx, vstr)
	if err != nil {
		return err
	}

	return nil
}

// Finalize proxy call performs both Finalize and GetProvisioners grpc calls.
func (e *executor) Finalize(ctx context.Context, calls []ContractCall, stateRoot []byte, height uint64, blockGasLiit uint64) (user.Provisioners, []byte, error) {
	vstr := new(rusk.StateTransitionRequest)
	vstr.Txs = make([]*rusk.Transaction, len(calls))
	vstr.BlockHeight = height
	vstr.BlockGasLimit = blockGasLiit

	for i, call := range calls {
		tx := new(rusk.Transaction)
		if err := MTransaction(tx, call.(*Transaction)); err != nil {
			return user.Provisioners{}, nil, err
		}

		vstr.Txs[i] = tx
	}

	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(e.txTimeout))
	defer cancel()

	res, err := e.stateClient.Finalize(ctx, vstr)
	if err != nil {
		return user.Provisioners{}, nil, err
	}

	provisioners := user.NewProvisioners()
	memberMap := make(map[string]*user.Member)

	pres, err := e.stateClient.GetProvisioners(ctx, &rusk.GetProvisionersRequest{})
	if err != nil {
		return user.Provisioners{}, nil, err
	}

	for i := range pres.Provisioners {
		member := new(user.Member)
		UMember(pres.Provisioners[i], member)
		memberMap[string(member.PublicKeyBLS)] = member
		provisioners.Set.Insert(member.PublicKeyBLS)
	}

	provisioners.Members = memberMap
	return *provisioners, res.StateRoot, nil
}

// Accept proxy call performs both Accept and GetProvisioners grpc calls.
func (e *executor) Accept(ctx context.Context, calls []ContractCall, stateRoot []byte, height, blockGasLimit uint64) (user.Provisioners, []byte, error) {
	vstr := new(rusk.StateTransitionRequest)
	vstr.Txs = make([]*rusk.Transaction, len(calls))
	vstr.BlockHeight = height
	vstr.BlockGasLimit = blockGasLimit

	for i, call := range calls {
		tx := new(rusk.Transaction)
		if err := MTransaction(tx, call.(*Transaction)); err != nil {
			return user.Provisioners{}, nil, err
		}

		vstr.Txs[i] = tx
	}

	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(e.txTimeout))
	defer cancel()

	res, err := e.stateClient.Accept(ctx, vstr)
	if err != nil {
		return user.Provisioners{}, nil, err
	}

	provisioners := user.NewProvisioners()
	memberMap := make(map[string]*user.Member)

	pres, err := e.stateClient.GetProvisioners(ctx, &rusk.GetProvisionersRequest{})
	if err != nil {
		return user.Provisioners{}, nil, err
	}

	for i := range pres.Provisioners {
		member := new(user.Member)
		UMember(pres.Provisioners[i], member)
		memberMap[string(member.PublicKeyBLS)] = member
		provisioners.Set.Insert(member.PublicKeyBLS)
	}

	provisioners.Members = memberMap
	return *provisioners, res.StateRoot, nil
}

// ExecuteStateTransition proxy call performs a single grpc ExecuteStateTransition call.
func (e *executor) ExecuteStateTransition(ctx context.Context, calls []ContractCall, blockGasLimit, blockHeight uint64) ([]ContractCall, []byte, error) {
	vstr := new(rusk.ExecuteStateTransitionRequest)
	vstr.Txs = make([]*rusk.Transaction, len(calls))
	vstr.BlockHeight = blockHeight
	vstr.BlockGasLimit = blockGasLimit

	for i, call := range calls {
		tx := new(rusk.Transaction)
		if err := MTransaction(tx, call.(*Transaction)); err != nil {
			return nil, nil, err
		}

		vstr.Txs[i] = tx
	}

	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(e.txTimeout))
	defer cancel()

	res, err := e.stateClient.ExecuteStateTransition(ctx, vstr)
	if err != nil {
		return nil, nil, err
	}

	if !res.Success {
		return nil, nil, errors.New("unsuccessful state transition function execution")
	}

	validCalls := make([]ContractCall, 0)

	for _, tx := range res.Txs {
		trans := NewTransaction()
		copy(trans.Hash[:], tx.GetTxHash())

		if err := UTransaction(tx.Tx, trans); err != nil {
			return nil, nil, err
		}

		validCalls = append(validCalls, trans)
	}

	return validCalls, res.StateRoot, nil
}

// GetProvisioners see also Executor.GetProvisioners.
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

// GetStateRoot proxy call to state.GetStateRoot grpc.
func (e *executor) GetStateRoot(ctx context.Context) ([]byte, error) {
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(e.txTimeout))
	defer cancel()

	r, err := e.stateClient.GetStateRoot(ctx, &rusk.GetStateRootRequest{})
	if err != nil {
		return nil, err
	}

	return r.StateRoot, nil
}

// UMember deep copies from the rusk.Provisioner.
func UMember(r *rusk.Provisioner, t *user.Member) {
	t.PublicKeyBLS = make([]byte, len(r.PublicKeyBls))
	copy(t.PublicKeyBLS, r.PublicKeyBls)

	t.RawPublicKeyBLS = make([]byte, len(r.RawPublicKeyBls))
	copy(t.RawPublicKeyBLS, r.RawPublicKeyBls)

	t.Stakes = make([]user.Stake, len(r.Stakes))
	for i := range r.Stakes {
		t.Stakes[i] = user.Stake{
			Amount:      r.Stakes[i].Amount,
			StartHeight: r.Stakes[i].StartHeight,
			EndHeight:   r.Stakes[i].EndHeight,
		}
	}
}
