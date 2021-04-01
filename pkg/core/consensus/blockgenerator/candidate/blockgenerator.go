// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package candidate

import (
	"bytes"
	"context"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/keys"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	log "github.com/sirupsen/logrus"
)

var lg = log.WithField("process", "candidate generator")

// MaxTxSetSize defines the maximum amount of transactions.
// It is TBD along with block size and processing.MaxFrameSize.
const MaxTxSetSize = 825000

// Generator is responsible for generating candidate blocks, and propagating them
// alongside received Scores. It is triggered by the ScoreEvent, sent by the score generator.
type Generator interface {
	GenerateCandidateMessage(ctx context.Context, sev message.ScoreProposal, r consensus.RoundUpdate, step uint8) (*message.Score, error)
}

type generator struct {
	*consensus.Emitter
	genPubKey *keys.PublicKey
}

// New creates a new block generator.
func New(e *consensus.Emitter, genPubKey *keys.PublicKey) Generator {
	return &generator{
		Emitter:   e,
		genPubKey: genPubKey,
	}
}

func (bg *generator) regenerateCommittee(r consensus.RoundUpdate) [][]byte {
	size := r.P.SubsetSizeAt(r.Round - 1)
	if size > agreement.MaxCommitteeSize {
		size = agreement.MaxCommitteeSize
	}

	return r.P.CreateVotingCommittee(r.Round-1, r.LastCertificate.Step, size).MemberKeys()
}

// PropagateBlockAndScore runs the generation of a `Score` and a candidate `block.Block`.
// The Generator will propagate both the Score and Candidate messages at the end
// of this function call.
func (bg *generator) GenerateCandidateMessage(ctx context.Context, sev message.ScoreProposal, r consensus.RoundUpdate, step uint8) (*message.Score, error) {
	log := lg.
		WithField("round", sev.State().Round).
		WithField("step", sev.State().Step)

	committee := bg.regenerateCommittee(r)

	blk, err := bg.Generate(sev, committee, r)
	if err != nil {
		log.
			WithError(err).
			Error("failed to bg.Generate")
		return nil, err
	}

	// Since the Candidate message goes straight to the Chain, there is
	// no need to use `SendAuthenticated`, as the header is irrelevant.
	// Thus, we will instead gossip it directly.
	return message.NewScore(sev, bg.Keys.BLSPubKeyBytes, r.Hash, *blk), nil
}

// Generate a Block.
func (bg *generator) Generate(sev message.ScoreProposal, keys [][]byte, r consensus.RoundUpdate) (*block.Block, error) {
	return bg.GenerateBlock(r.Round, sev.Seed, sev.Proof, sev.Score, r.Hash, keys)
}

// GenerateBlock generates a candidate block, by constructing the header and filling it
// with transactions from the mempool.
func (bg *generator) GenerateBlock(round uint64, seed, proof, score, prevBlockHash []byte, keys [][]byte) (*block.Block, error) {
	txs, err := bg.ConstructBlockTxs(proof, score, keys)
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

// ConstructBlockTxs will fetch all valid transactions from the mempool, append a coinbase
// transaction, and return them all.
func (bg *generator) ConstructBlockTxs(proof, score []byte, keys [][]byte) ([]transactions.ContractCall, error) {
	txs := make([]transactions.ContractCall, 0)

	// Retrieve and append the verified transactions from Mempool
	// Max transaction size param
	param := new(bytes.Buffer)
	if err := encoding.WriteUint32LE(param, uint32(MaxTxSetSize)); err != nil {
		return nil, err
	}

	timeoutGetMempoolTXsBySize := time.Duration(config.Get().Timeout.TimeoutGetMempoolTXsBySize) * time.Second
	resp, err := bg.RPCBus.Call(topics.GetMempoolTxsBySize, rpcbus.NewRequest(*param), timeoutGetMempoolTXsBySize)
	// TODO: GetVerifiedTxs should ensure once again that none of the txs have been
	// already accepted in the chain.
	if err != nil {
		return nil, err
	}

	txs = append(txs, resp.([]transactions.ContractCall)...)

	// Construct and append coinbase Tx to reward the generator
	// XXX: this needs to be adjusted
	coinbaseTx := transactions.RandDistributeTx(config.GeneratorReward, len(keys))
	txs = append(txs, coinbaseTx)

	return txs, nil
}
