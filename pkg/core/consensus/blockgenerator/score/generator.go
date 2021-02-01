// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package score

import (
	"context"
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/blindbid"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-crypto/bls"
	log "github.com/sirupsen/logrus"
)

var (
	emptyHash [32]byte
	lg        = log.WithField("process", "score blockGenerator")
)

// Generator provides the operation of calcularing a score for the candidate.
type Generator interface {
	// Generator
	Generate(ctx context.Context, r consensus.RoundUpdate, step uint8) message.ScoreProposal
}

type generator struct {
	*consensus.Emitter
	d              []byte
	k              []byte
	indexStoredBid uint64

	// d, edPk []byte
	scoreGenerator transactions.BlockGenerator

	lock      sync.Mutex
	threshold *consensus.Threshold
}

// New creates a new score generation step.
func New(e *consensus.Emitter, db database.DB) (Generator, error) {
	var d, k []byte
	var indexStoredBid uint64

	if err := db.View(func(t database.Transaction) error {
		var err error
		d, k, indexStoredBid, err = t.FetchBidValues()
		return err
	}); err != nil {
		return nil, err
	}

	return &generator{
		Emitter:        e,
		scoreGenerator: e.Proxy.BlockGenerator(),
		d:              d,
		k:              k,
		threshold:      consensus.NewThreshold(),
		indexStoredBid: indexStoredBid,
	}, nil
}

func (p *generator) sign(seed []byte) ([]byte, error) {
	signedSeed, err := bls.Sign(p.Keys.BLSSecretKey, p.Keys.BLSPubKey, seed)
	if err != nil {
		return nil, err
	}

	compSeed := signedSeed.Compress()
	return compSeed, nil
}

// Generate the score and if it is above the treshold pass it to the block
// generator for propagating it to the network.
func (p *generator) Generate(ctx context.Context, r consensus.RoundUpdate, step uint8) message.ScoreProposal {
	// TODO: check if we are in the BidList from RUSK. If we are not, we should
	// return immediately
	hdr := header.Header{
		Round:     r.Round,
		Step:      step,
		PubKeyBLS: p.Keys.BLSPubKeyBytes,
		BlockHash: emptyHash[:],
	}

	seed, err := p.sign(r.Seed)
	if err != nil {
		// TODO: this probably deserves a panic
		lg.WithError(err).Errorln("problem in signing the seed during the generation")
		return message.EmptyScoreProposal(hdr)
	}

	sr := blindbid.GenerateScoreRequest{
		K:              p.k,
		Seed:           seed,
		Secret:         p.d,
		Round:          uint32(r.Round),
		Step:           uint32(step),
		IndexStoredBid: p.indexStoredBid,
	}

	scoreTx, err := p.scoreGenerator.GenerateScore(ctx, sr)
	// GenerateScore would return error if we are not in this round bidlist, or
	// if the BidTransaction expired or is malformed
	if err != nil {
		lg.WithError(err).Errorln("problem in generating the score")
		return message.EmptyScoreProposal(hdr)
	}

	// This lock protects the threshold in the unlike case of two score
	// generations running at the same time. It should not happen, but we
	// cannot guarantee it. Hence the locking
	p.lock.Lock()
	if p.threshold.Exceeds(scoreTx.Score) {
		// TODO: log the error
		// return errors.New("proof score is below threshold")
		p.lock.Unlock()
		return message.EmptyScoreProposal(hdr)
	}
	p.lock.Unlock()

	return message.NewScoreProposal(hdr, seed, scoreTx)
}
