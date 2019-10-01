package generation

import (
	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	zkproof "github.com/dusk-network/dusk-zkproof"
	log "github.com/sirupsen/logrus"
)

// Generator defines the capability of generating proofs of blind bid, needed to
// propose blocks in the consensus.
type Generator interface {
	GenerateProof([]byte, user.BidList) zkproof.ZkProof
	InBidList(user.BidList) bool
	UpdateProofValues(ristretto.Scalar, ristretto.Scalar)
}

type proofGenerator struct {
	d, k, x ristretto.Scalar
}

func newProofGenerator(k ristretto.Scalar) (*proofGenerator, error) {
	return &proofGenerator{
		k: k,
	}, nil
}

func (g *proofGenerator) InBidList(bidList user.BidList) bool {
	var bid user.Bid
	copy(bid.X[:], g.x.Bytes())
	return bidList.Contains(bid)
}

func (g *proofGenerator) UpdateProofValues(d, m ristretto.Scalar) {
	x := zkproof.CalculateX(d, m)
	g.x = x
	g.d = d
}

// GenerateProof will generate the proof of blind bid, needed to successfully
// propose a block to the voting committee.
func (g *proofGenerator) GenerateProof(seed []byte, bidList user.BidList) zkproof.ZkProof {
	log.WithField("process", "generation").Traceln("generating proof")
	// Turn seed into scalar
	seedScalar := ristretto.Scalar{}
	seedScalar.Derive(seed)

	// Create a slice of scalars with a number of random bids (up to 10)
	bidListSubset := createBidListSubset(bidList)
	bidListScalars := convertBidListToScalars(bidListSubset)

	return zkproof.Prove(g.d, g.k, seedScalar, bidListScalars)
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
