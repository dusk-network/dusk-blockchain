package score

import (
	"bytes"
	"errors"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-crypto/bls"
	"github.com/dusk-network/dusk-wallet/key"
	zkproof "github.com/dusk-network/dusk-zkproof"
	log "github.com/sirupsen/logrus"
)

var _ consensus.Component = (*Generator)(nil)

var emptyHash [32]byte
var lg *log.Entry = log.WithField("process", "score generator")

func NewComponent(publisher eventbus.Publisher, consensusKeys key.ConsensusKeys, d, k ristretto.Scalar) *Generator {
	return &Generator{
		publisher:     publisher,
		ConsensusKeys: consensusKeys,
		k:             k,
		d:             d,
		threshold:     consensus.NewThreshold(),
	}
}

type Generator struct {
	publisher eventbus.Publisher
	roundInfo consensus.RoundUpdate
	seed      []byte
	d, k      ristretto.Scalar
	key.ConsensusKeys
	threshold *consensus.Threshold

	signer       consensus.Signer
	generationID uint32
}

func (g *Generator) Initialize(eventPlayer consensus.EventPlayer, signer consensus.Signer, ru consensus.RoundUpdate) []consensus.TopicListener {
	g.signer = signer
	g.roundInfo = ru
	signedSeed, err := g.sign(ru.Seed)
	if err != nil {
		lg.WithField("category", "BUG").WithError(err).Errorln("could not sign seed")
		return nil
	}

	g.seed = signedSeed

	// If we are not in this round's bid list, we can skip initialization, as there
	// would be no need to listen for these events if we are not qualified to generate
	// scores and blocks.
	if !inBidList(g.d, g.k, g.roundInfo.BidList) {
		return nil
	}

	generationSubscriber := consensus.TopicListener{
		Topic:    topics.Generation,
		Listener: consensus.NewSimpleListener(g.Collect, consensus.LowPriority, false),
	}
	g.generationID = generationSubscriber.Listener.ID()

	return []consensus.TopicListener{generationSubscriber}
}

func (g *Generator) ID() uint32 {
	return g.generationID
}

// Finalize implements consensus.Component.
func (g *Generator) Finalize() {}

// Prove will generate the proof of blind bid, needed to successfully
// propose a block to the voting committee.
func (g *Generator) Prove(seed []byte, bidList user.BidList) zkproof.ZkProof {
	log.Traceln("generating proof")
	// Turn seed into scalar
	seedScalar := ristretto.Scalar{}
	seedScalar.Derive(seed)

	// Create a slice of scalars with a number of random bids (up to 10)
	bidListSubset := createBidListSubset(bidList)
	bidListScalars := convertBidListToScalars(bidListSubset)

	return zkproof.Prove(g.d, g.k, seedScalar, bidListScalars)
}

func (g *Generator) Collect(e consensus.Event) error {
	defer g.threshold.Lower()
	return g.generateScore()
}

func (g *Generator) generateScore() error {
	proof := g.Prove(g.seed, g.roundInfo.BidList)
	if g.threshold.Exceeds(proof.Score) {
		return errors.New("proof score is below threshold")
	}

	sev := g.createScoreEvent(g.seed, proof)
	buf := new(bytes.Buffer)
	if err := Marshal(buf, sev); err != nil {
		return err
	}

	return g.signer.SendWithHeader(topics.ScoreEvent, emptyHash[:], buf, g.ID())
}

func (g *Generator) createScoreEvent(seed []byte, proof zkproof.ZkProof) Event {
	return Event{
		Proof: zkproof.ZkProof{
			Score:         proof.Score,
			Proof:         proof.Proof,
			Z:             proof.Z,
			BinaryBidList: proof.BinaryBidList,
		},
		Seed: seed,
	}
}

// bidsToScalars will take a global public list, take a subset from it, and then
// return it as a slice of scalars.
func createBidListSubset(bidList user.BidList) user.BidList {
	numBids := getNumBids(bidList)
	return bidList.Subset(numBids)
}

// getNumBids will return how many bids to include in the bid list subset
// for the proof.
func getNumBids(bidList user.BidList) int {
	numBids := len(bidList)
	if numBids > 10 {
		numBids = 10
	}

	return numBids
}

// convertBidListToScalars will take a BidList, and create a slice of scalars from it.
func convertBidListToScalars(bidList user.BidList) []ristretto.Scalar {
	scalarList := make([]ristretto.Scalar, len(bidList))
	for i, bid := range bidList {
		bidScalar := ristretto.Scalar{}
		err := bidScalar.UnmarshalBinary(bid.X[:])
		if err != nil {
			log.WithError(err).WithField("process", "proofgenerator").Errorln("Error in converting Bid List to scalar")
			panic(err)
		}
		scalarList[i] = bidScalar
	}

	return scalarList
}

func inBidList(d, k ristretto.Scalar, bidList user.BidList) bool {
	m := zkproof.CalculateM(k)
	x := zkproof.CalculateX(d, m)
	var bid user.Bid
	copy(bid.X[:], x.Bytes())
	return bidList.Contains(bid)
}

func (g *Generator) sign(seed []byte) ([]byte, error) {
	signedSeed, err := bls.Sign(g.ConsensusKeys.BLSSecretKey, g.ConsensusKeys.BLSPubKey, seed)
	if err != nil {
		return nil, err
	}
	compSeed := signedSeed.Compress()
	return compSeed, nil
}
