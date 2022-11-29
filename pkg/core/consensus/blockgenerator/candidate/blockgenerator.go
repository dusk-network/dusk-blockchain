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

var errEmptyStateHash = errors.New("empty state hash")

// Generator is responsible for generating candidate blocks, and propagating them
// alongside received Scores. It is triggered by the ScoreEvent, sent by the score generator.
type Generator interface {
	GenerateCandidateMessage(ctx context.Context, r consensus.RoundUpdate, step uint8) (*message.NewBlock, error)
}

type generator struct {
	*consensus.Emitter

	callTimeout time.Duration
	executeFn   consensus.ExecuteTxsFunc
}

// New creates a new block generator.
func New(e *consensus.Emitter, executeFn consensus.ExecuteTxsFunc) Generator {
	ct := config.Get().Timeout.TimeoutGetMempoolTXsBySize
	if ct == 0 {
		ct = 5
	}

	return &generator{
		Emitter:     e,
		executeFn:   executeFn,
		callTimeout: time.Duration(ct) * time.Second,
	}
}

// PropagateBlockAndScore runs the generation of a `Score` and a candidate `block.Block`.
// The Generator will propagate both the Score and Candidate messages at the end
// of this function call.
func (bg *generator) GenerateCandidateMessage(ctx context.Context, r consensus.RoundUpdate, step uint8) (*message.NewBlock, error) {
	log := lg.
		WithField("round", r.Round).
		WithField("step", step)

	seed, err := bg.sign(r.Seed)
	if err != nil {
		return nil, err
	}

	blk, err := bg.Generate(ctx, seed, r)
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
func (bg *generator) Generate(ctx context.Context, seed []byte, r consensus.RoundUpdate) (*block.Block, error) {
	return bg.GenerateBlock(ctx, r.Round, seed, r.Hash, r.Timestamp)
}

func (bg *generator) execute(ctx context.Context, txs []transactions.ContractCall, round uint64, gasLimit uint64) ([]transactions.ContractCall, []byte, error) {
	txs, stateHash, err := bg.executeFn(ctx, txs, round, gasLimit, bg.Keys.BLSPubKey)
	if err != nil {
		return nil, nil, err
	}

	if len(stateHash) == 0 {
		return nil, nil, errEmptyStateHash
	}

	return txs, stateHash, nil
}

// fetchOrTimeout will keep trying to FetchMempoolTxs() until either
// we get some txs or the timeout expires.
func (bg *generator) fetchOrTimeout(ctx context.Context) ([]transactions.ContractCall, error) {
	delay := config.Get().Mempool.ExtractionDelaySecs
	if delay == 0 || config.Get().Consensus.ConsensusTimeOut < delay {
		return bg.FetchMempoolTxs()
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(delay)*time.Second)
	defer cancel()

	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return bg.FetchMempoolTxs()
		case <-tick.C:
			txs, err := bg.FetchMempoolTxs()
			if err != nil {
				return nil, err
			}

			if len(txs) > 0 {
				return txs, nil
			}
		}
	}
}

// GenerateBlock generates a candidate block, by constructing the header and filling it
// with transactions from the mempool.
func (bg *generator) GenerateBlock(ctx context.Context, round uint64, seed, prevBlockHash []byte, prevBlockTimestamp int64) (*block.Block, error) {
	txs, err := bg.fetchOrTimeout(ctx)
	if err != nil {
		return nil, err
	}

	blockGasLimit := config.Get().State.BlockGasLimit

	txs, stateHash, err := bg.execute(context.Background(), txs, round, blockGasLimit)
	if err != nil {
		return nil, err
	}

	timestamp := time.Now().Unix()
	maxTimestamp := prevBlockTimestamp + config.MaxBlockTime

	if round > 1 && prevBlockTimestamp > 0 {
		if timestamp < prevBlockTimestamp {
			timestamp = prevBlockTimestamp
		} else if timestamp > maxTimestamp {
			// block time should not exceed config.MaxBlockTime
			timestamp = maxTimestamp
		}
	}

	// Construct header
	h := &block.Header{
		Version:            0,
		Timestamp:          timestamp,
		Height:             round,
		PrevBlockHash:      prevBlockHash,
		GeneratorBlsPubkey: bg.Keys.BLSPubKey,
		Seed:               seed,
		Certificate:        block.EmptyCertificate(),
		StateHash:          stateHash,
		GasLimit:           blockGasLimit,
	}

	// Construct the candidate block
	candidateBlock := &block.Block{
		Header: h,
		Txs:    txs,
	}

	// Generate the block hash
	hash, err := candidateBlock.CalculateHash()
	if err != nil {
		return nil, err
	}

	candidateBlock.Header.Hash = hash
	return candidateBlock, nil
}

// FetchMempoolTxs will fetch all valid transactions from the mempool.
func (bg *generator) FetchMempoolTxs() ([]transactions.ContractCall, error) {
	// Retrieve and append the verified transactions from Mempool
	// Max transaction size param
	param := new(bytes.Buffer)
	if err := encoding.WriteUint32LE(param, config.MaxTxSetSize); err != nil {
		return nil, err
	}

	resp, err := bg.RPCBus.Call(topics.GetMempoolTxsBySize, rpcbus.NewRequest(*param), bg.callTimeout)
	if err != nil {
		return nil, err
	}

	return resp.([]transactions.ContractCall), nil
}

func (bg *generator) sign(seed []byte) ([]byte, error) {
	return bls.Sign(bg.Keys.BLSSecretKey, bg.Keys.BLSPubKey, seed)
}
