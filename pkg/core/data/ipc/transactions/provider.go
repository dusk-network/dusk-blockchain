// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package transactions

import (
	"context"
	"encoding/hex"
	"errors"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/metadata"
)

// UnconfirmedTxProber performs verification of contract calls (transactions).
type UnconfirmedTxProber interface {
	// VerifyTransaction verifies a contract call transaction.
	Preverify(context.Context, ContractCall) ([]byte, Fee, error)
}

// Executor encapsulate the Global State operations.
type Executor interface {
	// VerifyStateTransition performs dry-run state transition to ensure all txs are valid.
	VerifyStateTransition(context.Context, []ContractCall, uint64, uint64, []byte) ([]byte, error)

	// ExecuteStateTransition performs dry-run state transition to return valid-only set of txs and state hash.
	ExecuteStateTransition(context.Context, []ContractCall, uint64, uint64, []byte) ([]ContractCall, []byte, error)

	// Accept creates an ephemeral state transition.
	Accept(context.Context, []ContractCall, []byte, uint64, uint64, []byte) ([]ContractCall, user.Provisioners, []byte, error)

	// Finalize creates a finalized state transition.
	Finalize(context.Context, []ContractCall, []byte, uint64, uint64, []byte) ([]ContractCall, user.Provisioners, []byte, error)

	// GetProvisioners returns the current set of provisioners.
	GetProvisioners(ctx context.Context) (user.Provisioners, error)

	// GetStateRoot returns root hash of the finalized state.
	GetStateRoot(ctx context.Context) ([]byte, error)

	// Persist instructs Rusk to persist the state if in-sync with provided stateRoot.
	Persist(ctx context.Context, stateRoot []byte) error

	// Revert instructs Rusk to revert to the most recent finalized state. Returns stateRoot, if no error.
	Revert(ctx context.Context) ([]byte, error)
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
	decoded, decodeErr := call.Decode()
	if decodeErr != nil {
		return nil, Fee{}, decodeErr
	}

	if decoded.Fee.GasLimit >= config.Get().State.BlockGasLimit {
		return nil, Fee{}, errors.New("tx gas limit exceeds the block gas limit")
	}

	vstr := new(rusk.PreverifyRequest)

	tx := new(rusk.Transaction)
	if err := MTransaction(tx, call.(*Transaction)); err != nil {
		return nil, Fee{}, err
	}

	vstr.Tx = tx

	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(v.txTimeout))
	defer cancel()

	ruskCtx := injectRuskVersion(ctx)

	res, err := v.stateClient.Preverify(ruskCtx, vstr)
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
func (e *executor) VerifyStateTransition(ctx context.Context, calls []ContractCall, blockGasLimit, blockHeight uint64, generator []byte) ([]byte, error) {
	vstr := new(rusk.VerifyStateTransitionRequest)
	vstr.Txs = make([]*rusk.Transaction, len(calls))
	vstr.Generator = generator

	for i, call := range calls {
		tx := new(rusk.Transaction)
		if err := MTransaction(tx, call.(*Transaction)); err != nil {
			return nil, err
		}

		vstr.Txs[i] = tx
	}

	vstr.BlockHeight = blockHeight
	vstr.BlockGasLimit = blockGasLimit

	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(e.txTimeout))
	defer cancel()

	ruskCtx := injectRuskVersion(ctx)

	resp, err := e.stateClient.VerifyStateTransition(ruskCtx, vstr)
	if err != nil {
		return nil, err
	}

	return resp.GetStateRoot(), nil
}

// Finalize proxy call performs both Finalize and GetProvisioners grpc calls.
func (e *executor) Finalize(ctx context.Context, calls []ContractCall, stateRoot []byte, height uint64, blockGasLimit uint64, generator []byte) ([]ContractCall, user.Provisioners, []byte, error) {
	vstr := new(rusk.StateTransitionRequest)
	vstr.Txs = make([]*rusk.Transaction, len(calls))
	vstr.BlockHeight = height
	vstr.BlockGasLimit = blockGasLimit
	vstr.Generator = generator

	for i, call := range calls {
		tx := new(rusk.Transaction)
		if err := MTransaction(tx, call.(*Transaction)); err != nil {
			return nil, user.Provisioners{}, nil, err
		}

		vstr.Txs[i] = tx
	}

	ruskCtx := injectRuskVersion(ctx)

	// No deadline grpc call. It's all or nothing. This is to avoid a scenario
	// where grpc call returns DeadlineExceeded error which may end up into an
	// inconsistency between rusk and dusk service states.
	res, err := e.stateClient.Finalize(ruskCtx, vstr)
	if err != nil {
		return nil, user.Provisioners{}, nil, err
	}

	provisioners := user.NewProvisioners()
	memberMap := make(map[string]*user.Member)

	pres, err := e.stateClient.GetProvisioners(ruskCtx, &rusk.GetProvisionersRequest{})
	if err != nil {
		return nil, user.Provisioners{}, nil, err
	}

	for i := range pres.Provisioners {
		member := new(user.Member)
		UMember(pres.Provisioners[i], member)
		memberMap[string(member.PublicKeyBLS)] = member
		provisioners.Set.Insert(member.PublicKeyBLS)
	}

	resCalls, err := e.convertToContractCall(res.Txs)
	if err != nil {
		return nil, user.Provisioners{}, nil, err
	}

	provisioners.Members = memberMap

	return resCalls, *provisioners, res.StateRoot, nil
}

// Accept proxy call performs both Accept and GetProvisioners grpc calls.
func (e *executor) Accept(ctx context.Context, calls []ContractCall, stateRoot []byte, height, blockGasLimit uint64, generator []byte) ([]ContractCall, user.Provisioners, []byte, error) {
	vstr := new(rusk.StateTransitionRequest)
	vstr.Txs = make([]*rusk.Transaction, len(calls))
	vstr.BlockHeight = height
	vstr.BlockGasLimit = blockGasLimit
	vstr.Generator = generator

	for i, call := range calls {
		tx := new(rusk.Transaction)
		if err := MTransaction(tx, call.(*Transaction)); err != nil {
			return nil, user.Provisioners{}, nil, err
		}

		vstr.Txs[i] = tx
	}

	ruskCtx := injectRuskVersion(ctx)

	// No deadline grpc call. It's all or nothing. This is to avoid a scenario
	// where grpc call returns DeadlineExceeded error which may end up into an
	// inconsistency between rusk and dusk service states.
	res, err := e.stateClient.Accept(ruskCtx, vstr)
	if err != nil {
		return nil, user.Provisioners{}, nil, err
	}

	provisioners := user.NewProvisioners()
	memberMap := make(map[string]*user.Member)

	pres, err := e.stateClient.GetProvisioners(ruskCtx, &rusk.GetProvisionersRequest{})
	if err != nil {
		return nil, user.Provisioners{}, nil, err
	}

	for i := range pres.Provisioners {
		member := new(user.Member)
		UMember(pres.Provisioners[i], member)
		memberMap[string(member.PublicKeyBLS)] = member
		provisioners.Set.Insert(member.PublicKeyBLS)
	}

	resCalls, err := e.convertToContractCall(res.Txs)
	if err != nil {
		return nil, user.Provisioners{}, nil, err
	}

	provisioners.Members = memberMap

	return resCalls, *provisioners, res.StateRoot, nil
}

// ExecuteStateTransition proxy call performs a single grpc ExecuteStateTransition call.
func (e *executor) ExecuteStateTransition(ctx context.Context, calls []ContractCall, blockGasLimit, blockHeight uint64, generator []byte) ([]ContractCall, []byte, error) {
	vstr := new(rusk.ExecuteStateTransitionRequest)
	vstr.Txs = make([]*rusk.Transaction, len(calls))
	vstr.BlockHeight = blockHeight
	vstr.BlockGasLimit = blockGasLimit
	vstr.Generator = generator

	for i, call := range calls {
		tx := new(rusk.Transaction)
		if err := MTransaction(tx, call.(*Transaction)); err != nil {
			return nil, nil, err
		}

		vstr.Txs[i] = tx
	}

	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(e.txTimeout))
	defer cancel()

	ruskCtx := injectRuskVersion(ctx)

	res, err := e.stateClient.ExecuteStateTransition(ruskCtx, vstr)
	if err != nil {
		return nil, nil, err
	}

	resCalls, err := e.convertToContractCall(res.Txs)
	if err != nil {
		return nil, nil, err
	}

	return resCalls, res.StateRoot, nil
}

// convertToContractCall converts a set of ExecutedTransaction transactions into set of ContractCall transactions.
func (e *executor) convertToContractCall(txs []*rusk.ExecutedTransaction) ([]ContractCall, error) {
	res := make([]ContractCall, 0)

	for _, tx := range txs {
		cc := NewTransaction()
		copy(cc.Hash[:], tx.GetTxHash())
		cc.GasSpentValue = tx.GetGasSpent()
		cc.Error = tx.GetError()

		logrus.WithField("txid", hex.EncodeToString(tx.GetTxHash())).
			WithField("gas_spent", tx.GetGasSpent()).
			WithField("txtype", tx.Tx.GetType()).
			Trace("rusk transaction")

		if err := UTransaction(tx.Tx, cc); err != nil {
			return nil, err
		}

		res = append(res, cc)
	}

	return res, nil
}

// GetProvisioners see also Executor.GetProvisioners.
func (e *executor) GetProvisioners(ctx context.Context) (user.Provisioners, error) {
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(e.txTimeout))
	defer cancel()

	provisioners := user.NewProvisioners()
	memberMap := make(map[string]*user.Member)
	ruskCtx := injectRuskVersion(ctx)

	pres, err := e.stateClient.GetProvisioners(ruskCtx, &rusk.GetProvisionersRequest{})
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

	ruskCtx := injectRuskVersion(ctx)

	r, err := e.stateClient.GetStateRoot(ruskCtx, &rusk.GetStateRootRequest{})
	if err != nil {
		return nil, err
	}

	return r.StateRoot, nil
}

// Persist proxy call to state.Persist grpc.
func (e *executor) Persist(ctx context.Context, stateRoot []byte) error {
	req := &rusk.PersistRequest{StateRoot: stateRoot}
	ruskCtx := injectRuskVersion(ctx)

	// No deadline grpc call. It's all or nothing. This is to avoid a scenario
	// where grpc call returns DeadlineExceeded error which may end up into an
	// inconsistency between rusk and dusk service states.
	_, err := e.stateClient.Persist(ruskCtx, req)
	if err != nil {
		return err
	}

	return nil
}

// Revert to the last finalized state with a call to the client.
func (e *executor) Revert(ctx context.Context) ([]byte, error) {
	req := &rusk.RevertRequest{}
	ruskCtx := injectRuskVersion(ctx)

	resp, err := e.stateClient.Revert(ruskCtx, req)
	if err != nil {
		return nil, err
	}

	return resp.StateRoot, nil
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
			Value:       r.Stakes[i].Value,
			Reward:      r.Stakes[i].Reward,
			Counter:     r.Stakes[i].Counter,
			Eligibility: r.Stakes[i].Eligibility,
		}
	}
}

func injectRuskVersion(ctx context.Context) context.Context {
	md := metadata.New(map[string]string{"x-rusk-version": config.RuskVersion})
	return metadata.NewOutgoingContext(ctx, md)
}
