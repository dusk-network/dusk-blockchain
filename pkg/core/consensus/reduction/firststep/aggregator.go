package firststep

import (
	"bytes"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/sortedset"
	log "github.com/sirupsen/logrus"
)

// The aggregator acts as a de facto storage unit for Reduction messages. Any message
// it receives will be aggregated into a StepVotes struct, organized by block hash.
// Once the key set for a StepVotes of a certain block hash reaches quorum, this
// StepVotes is passed on to the Reducer by use of the `haltChan` channel.
// An aggregator should be instantiated on a per-step basis and is no longer usable
// after reaching quorum and sending on `haltChan`.
type aggregator struct {
	handler *reduction.Handler
	rpcBus  *rpcbus.RPCBus

	voteSets map[string]struct {
		*message.StepVotes
		sortedset.Cluster
	}
}

// newAggregator returns an instantiated aggregator, ready for use.
func newAggregator(
	handler *reduction.Handler,
	rpcBus *rpcbus.RPCBus) *aggregator {
	return &aggregator{
		handler: handler,
		rpcBus:  rpcBus,
		voteSets: make(map[string]struct {
			*message.StepVotes
			sortedset.Cluster
		}),
	}
}

// Collect a Reduction message, and add its sender public key and signature to the
// StepVotes/Set kept under the corresponding block hash. If the Set reaches or exceeds
// quorum, the candidate block for the given block hash is first verified before
// propagating the information to the Reducer.
func (a *aggregator) collectVote(ev message.Reduction) (*result, error) {
	hdr := ev.State()
	hash := string(hdr.BlockHash)
	sv, found := a.voteSets[hash]
	if !found {
		sv.StepVotes = message.NewStepVotes()
		sv.Cluster = sortedset.NewCluster()
	}
	if err := sv.StepVotes.Add(ev.SignedHash, hdr.PubKeyBLS, hdr.Step); err != nil {
		lg.
			WithError(err).
			WithField("round", hdr.Round).
			WithField("step", hdr.Step).
			WithField("quorum", sv.Cluster.TotalOccurrences()).
			Debug("firststep, StepVotes.Add failed")
		return nil, err
	}

	votes := a.handler.VotesFor(hdr.PubKeyBLS, hdr.Round, hdr.Step)
	for i := 0; i < votes; i++ {
		sv.Cluster.Insert(hdr.PubKeyBLS)
	}
	a.voteSets[hash] = sv
	if sv.Cluster.TotalOccurrences() >= a.handler.Quorum(hdr.Round) {
		lg.
			WithField("round", hdr.Round).
			WithField("step", hdr.Step).
			WithField("quorum", sv.Cluster.TotalOccurrences()).
			Debug("firststep_quorum_reached")

		a.addBitSet(sv.StepVotes, sv.Cluster, hdr.Round, hdr.Step)

		blockHash := hdr.BlockHash

		// if the votes converged for an empty hash we invoke halt with no
		// StepVotes
		if !bytes.Equal(blockHash, emptyHash[:]) {
			if err := verifyCandidateBlock(a.rpcBus, blockHash); err != nil {
				log.
					WithError(err).
					WithField("round", hdr.Round).
					WithField("step", hdr.Step).
					Error("firststep_verifyCandidateBlock the candidate block failed")
				return &result{emptyHash[:], emptyStepVotes}, nil
			}
		}

		return &result{blockHash, *sv.StepVotes}, nil
	}

	lg.
		WithField("round", hdr.Round).
		WithField("step", hdr.Step).
		WithField("quorum", sv.Cluster.TotalOccurrences()).
		Debug("firststep_quorum_not_reached")
	return nil, nil
}

func (a *aggregator) addBitSet(sv *message.StepVotes, cluster sortedset.Cluster, round uint64, step uint8) {
	committee := a.handler.Committee(round, step)
	sv.BitSet = committee.Bits(cluster.Set)
}

func verifyCandidateBlock(rpcBus *rpcbus.RPCBus, blockHash []byte) error {
	// Fetch the candidate block first.

	params := new(bytes.Buffer)
	_ = encoding.Write256(params, blockHash)
	_ = encoding.WriteBool(params, true)

	req := rpcbus.NewRequest(*params)
	timeoutGetCandidate := time.Duration(config.Get().Timeout.TimeoutGetCandidate) * time.Second
	resp, err := rpcBus.Call(topics.GetCandidate, req, timeoutGetCandidate)
	if err != nil {
		log.
			WithError(err).
			WithFields(log.Fields{
				"process": "reduction",
			}).Error("firststep, fetching the candidate block failed")
		return err
	}
	cm := resp.(message.Candidate)

	// If our result was not a zero value hash, we should first verify it
	// before voting on it again
	if !bytes.Equal(blockHash, emptyHash[:]) {
		req := rpcbus.NewRequest(cm)
		timeoutVerifyCandidateBlock := time.Duration(config.Get().Timeout.TimeoutVerifyCandidateBlock) * time.Second
		if _, err := rpcBus.Call(topics.VerifyCandidateBlock, req, timeoutVerifyCandidateBlock); err != nil {
			log.
				WithError(err).
				WithFields(log.Fields{
					"process": "reduction",
				}).Error("firststep, verifying the candidate block failed")
			return err
		}
	}

	return nil
}
