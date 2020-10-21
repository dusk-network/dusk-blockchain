package score

import (
	"context"
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/candidate"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/blindbid"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/common"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-crypto/bls"
	log "github.com/sirupsen/logrus"
)

var emptyHash [32]byte
var lg = log.WithField("process", "score generator")

// Phase of the consensus
type Phase struct {
	*consensus.Emitter
	next           consensus.Phase
	bg             transactions.BlockGenerator
	d              *common.JubJubCompressed
	k              *common.BlsScalar
	indexStoredBid uint64

	//d, edPk []byte
	generator candidate.Generator

	lock      sync.Mutex
	threshold *consensus.Threshold
}

// New creates a new score generation step
func New(next consensus.Phase, e *consensus.Emitter, bg candidate.Generator) (*Phase, error) {
	var d, k []byte
	var indexStoredBid uint64
	_, db := heavy.CreateDBConnection()

	if err := db.View(func(t database.Transaction) error {
		var err error
		d, k, indexStoredBid, err = t.FetchBidValues()
		return err
	}); err != nil {
		return nil, err
	}

	return &Phase{
		Emitter:        e,
		bg:             e.Proxy.BlockGenerator(),
		d:              &common.JubJubCompressed{Data: d},
		k:              &common.BlsScalar{Data: k},
		threshold:      consensus.NewThreshold(),
		next:           next,
		generator:      bg,
		indexStoredBid: indexStoredBid,
	}, nil
}

// SetNext sets the next step to be returned at the end of this one
func (p *Phase) SetNext(next consensus.Phase) {
	p.next = next
}

// Fn returns the Phase state function
func (p *Phase) Fn(_ consensus.InternalPacket) consensus.PhaseFn {
	return p.Run
}

// Run the generation phase. This runs concurrently with the Selection phase
// and therefore we return the Selection phase immediately
func (p *Phase) Run(ctx context.Context, _ *consensus.Queue, _ chan message.Message, r consensus.RoundUpdate, step uint8) consensus.PhaseFn {
	go p.generate(ctx, r, step)

	// since the generation runs in parallel with the selection, we cannot
	// inject our own score and need to add it to the chan
	return p.next.Fn(nil)
}

func (p *Phase) sign(seed []byte) ([]byte, error) {
	signedSeed, err := bls.Sign(p.Keys.BLSSecretKey, p.Keys.BLSPubKey, seed)
	if err != nil {
		return nil, err
	}
	compSeed := signedSeed.Compress()
	return compSeed, nil
}

func (p *Phase) generate(ctx context.Context, r consensus.RoundUpdate, step uint8) {
	// TODO: check if we are in the BidList from RUSK. If we are not, we should
	// return immediately

	seed, err := p.sign(r.Seed)
	if err != nil {
		//TODO: this probably deserves a panic
		lg.WithError(err).Errorln("problem in signing the seed during the generation")
		return
	}

	sr := blindbid.GenerateScoreRequest{
		Commitment:     p.d,
		K:              p.k,
		Seed:           &common.BlsScalar{ Data: seed },
		Round:          uint32(r.Round),
		Step:           uint32(step),
		IndexStoredBid: p.indexStoredBid,
	}

	scoreTx, err := p.bg.GenerateScore(ctx, sr)
	// GenerateScore would return error if we are not in this round bidlist, or
	// if the BidTransaction expired or is malformed
	if err != nil {
		lg.WithError(err).Errorln("problem in generating the score")
		return
	}

	// This lock protects the threshold in the unlike case of two score
	// generations running at the same time. It should not happen, but we
	// cannot guarantee it. Hence the locking
	p.lock.Lock()
	if p.threshold.Exceeds(scoreTx.Score) {
		//TODO: log the error
		//return errors.New("proof score is below threshold")
		p.lock.Unlock()
		return
	}
	p.lock.Unlock()

	hdr := header.Header{
		Round:     r.Round,
		Step:      step,
		PubKeyBLS: p.Keys.BLSPubKeyBytes,
		BlockHash: emptyHash[:],
	}

	se := message.NewScoreProposal(hdr, seed, scoreTx)

	if err := p.generator.PropagateBlockAndScore(ctx, se, r, step); err != nil {
		lg.WithError(err).Errorln("candidate block generation failed")
	}
}
