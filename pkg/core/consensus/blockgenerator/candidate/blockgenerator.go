// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package candidate

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/dusk-network/bls12_381-sign/go/cgo/bls"
	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	log "github.com/sirupsen/logrus"
)

var lg = log.WithField("process", "consensus").WithField("actor", "candidate_generator")

var (
	errEmptyStateHash       = errors.New("empty state hash")
	errEmptyTxsList         = errors.New("empty list of transactions")
	errDistributeTxNotFound = errors.New("distribute tx not found")
)

// MaxTxSetSize defines the maximum amount of transactions.
// It is TBD along with block size and processing.MaxFrameSize.
const MaxTxSetSize = 825000

// Generator is responsible for generating candidate blocks, and propagating them
// alongside received Scores. It is triggered by the ScoreEvent, sent by the score generator.
type Generator interface {
	GenerateCandidateMessage(ctx context.Context, r consensus.RoundUpdate, step uint8) (*message.NewBlock, error)
}

type generator struct {
	*consensus.Emitter

	executeFn consensus.ExecuteTxsFunc
}

// New creates a new block generator.
func New(e *consensus.Emitter, executeFn consensus.ExecuteTxsFunc) Generator {
	return &generator{
		Emitter:   e,
		executeFn: executeFn,
	}
}

func (bg *generator) regenerateCommittee(r consensus.RoundUpdate) [][]byte {
	size := r.P.SubsetSizeAt(r.Round - 1)
	if size > agreement.MaxCommitteeSize {
		size = agreement.MaxCommitteeSize
	}

	return r.P.CreateVotingCommittee(r.Seed, r.Round-1, r.LastCertificate.Step, size).MemberKeys()
}

// PropagateBlockAndScore runs the generation of a `Score` and a candidate `block.Block`.
// The Generator will propagate both the Score and Candidate messages at the end
// of this function call.
func (bg *generator) GenerateCandidateMessage(ctx context.Context, r consensus.RoundUpdate, step uint8) (*message.NewBlock, error) {
	log := lg.
		WithField("round", r.Round).
		WithField("step", step)

	committee := bg.regenerateCommittee(r)

	seed, err := bg.sign(r.Seed)
	if err != nil {
		return nil, err
	}

	blk, err := bg.Generate(seed, committee, r)
	if err != nil {
		log.
			WithError(err).
			Error("failed to generate candidate block")
		return nil, err
	}

	hdr := header.Header{
		PubKeyBLS: bg.Keys.BLSPubKey,
		Round:     r.Round,
		Step:      step,
		BlockHash: blk.Header.Hash,
	}

	// Since the Candidate message goes straight to the Chain, there is
	// no need to use `SendAuthenticated`, as the header is irrelevant.
	// Thus, we will instead gossip it directly.
	scr := message.NewNewBlock(hdr, r.Hash, *blk)

	sig, err := bg.Sign(hdr)
	if err != nil {
		return nil, err
	}

	scr.SignedHash = sig
	return scr, nil
}

// Generate a Block.
func (bg *generator) Generate(seed []byte, keys [][]byte, r consensus.RoundUpdate) (*block.Block, error) {
	return bg.GenerateBlock(r.Round, seed, r.Hash, keys)
}

func (bg *generator) execute(ctx context.Context, txs []transactions.ContractCall, round uint64) ([]transactions.ContractCall, []byte, error) {
	txs, stateHash, err := bg.executeFn(ctx, txs, round)
	if err != nil {
		return nil, nil, err
	}

	// Ensure last item from returned txs is the Distribute tx
	if txs[len(txs)-1].Type() != transactions.Distribute {
		return nil, nil, errDistributeTxNotFound
	}

	if len(stateHash) == 0 {
		return nil, nil, errEmptyStateHash
	}

	return txs, stateHash, nil
}

// GenerateBlock generates a candidate block, by constructing the header and filling it
// with transactions from the mempool.
func (bg *generator) GenerateBlock(round uint64, seed, prevBlockHash []byte, keys [][]byte) (*block.Block, error) {
	txs, err := bg.FetchMempoolTxs(keys)
	if err != nil {
		return nil, err
	}

	txs, stateHash, err := bg.execute(context.Background(), txs, round)
	if err != nil {
		return nil, err
	}

	// Construct header
	h := &block.Header{
		Version:       0,
		Timestamp:     time.Now().Unix(),
		Height:        round,
		PrevBlockHash: prevBlockHash,
		TxRoot:        nil,
		Seed:          seed,
		Certificate:   block.EmptyCertificate(),
		StateHash:     stateHash,
	}

	// Construct the candidate block
	candidateBlock := &block.Block{
		Header: h,
		Txs:    txs,
	}

	// Update TxRoot
	root, err := candidateBlock.CalculateRoot()
	if err != nil {
		lg.
			WithError(err).
			Error("failed to CalculateRoot")
		return nil, err
	}

	candidateBlock.Header.TxRoot = root

	// Generate the block hash
	hash, err := candidateBlock.CalculateHash()
	if err != nil {
		return nil, err
	}

	candidateBlock.Header.Hash = hash
	return candidateBlock, nil
}

// FetchMempoolTxs will fetch all valid transactions from the mempool.
func (bg *generator) FetchMempoolTxs(keys [][]byte) ([]transactions.ContractCall, error) {
	// Retrieve and append the verified transactions from Mempool
	// Max transaction size param
	param := new(bytes.Buffer)
	if err := encoding.WriteUint32LE(param, uint32(MaxTxSetSize)); err != nil {
		return nil, err
	}

	timeoutGetMempoolTXsBySize := time.Duration(config.Get().Timeout.TimeoutGetMempoolTXsBySize) * time.Second
	resp, err := bg.RPCBus.Call(topics.GetMempoolTxsBySize, rpcbus.NewRequest(*param), timeoutGetMempoolTXsBySize)
	if err != nil {
		return nil, err
	}

	return resp.([]transactions.ContractCall), nil
}

func (bg *generator) sign(seed []byte) ([]byte, error) {
	return bls.Sign(bg.Keys.BLSSecretKey, bg.Keys.BLSPubKey, seed)
}
