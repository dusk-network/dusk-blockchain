package message

import (
	"bytes"
	"strings"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/util"
	"github.com/dusk-network/dusk-crypto/bls"
	"github.com/dusk-network/dusk-wallet/v2/key"
)

type (
	// Reduction is one of the messages used in the consensus algorithms. As
	// such it encapsulates a header.Header to allow the Coordinator to
	// correctly enforce the sequence of state changes expressed through the algorithm
	Reduction struct {
		hdr        header.Header
		SignedHash []byte
	}
)

// New returns and empty Reduction event.
func NewReduction(hdr header.Header) *Reduction {
	return &Reduction{
		hdr:        hdr,
		SignedHash: make([]byte, 33),
	}
}

func (r Reduction) String() string {
	var sb strings.Builder
	sb.WriteString(r.hdr.String())
	sb.WriteString(" signature' ")
	sb.WriteString(util.StringifyBytes(r.SignedHash))
	sb.WriteString("'")
	return sb.String()
}

// Header is used to comply to consensus.Message
func (r Reduction) State() header.Header {
	return r.hdr
}

// Header is used to comply to consensus.Message
func (r Reduction) Sender() []byte {
	return r.hdr.Sender()
}

// Equal is used to comply to consensus.Message
func (r Reduction) Equal(msg Message) bool {
	m, ok := msg.Payload().(Reduction)
	return ok && r.hdr.Equal(m.hdr) && bytes.Equal(r.SignedHash, m.SignedHash)
}

func UnmarshalReductionMessage(r *bytes.Buffer, m SerializableMessage) error {
	bev := NewReduction(header.Header{})
	if err := header.Unmarshal(r, &bev.hdr); err != nil {
		return err
	}

	if err := UnmarshalReduction(r, bev); err != nil {
		return err
	}

	m.SetPayload(*bev)
	return nil
}

// Unmarshal unmarshals the buffer into a Reduction event.
func UnmarshalReduction(r *bytes.Buffer, bev *Reduction) error {
	bev.SignedHash = make([]byte, 33)
	return encoding.ReadBLS(r, bev.SignedHash)
}

// Marshal a Reduction event into a buffer.
func MarshalReduction(r *bytes.Buffer, bev Reduction) error {
	if err := header.Marshal(r, bev.State()); err != nil {
		return err
	}

	if err := encoding.WriteBLS(r, bev.SignedHash); err != nil {
		return err
	}

	return nil
}

func UnmarshalVoteSet(r *bytes.Buffer) ([]Reduction, error) {
	length, err := encoding.ReadVarInt(r)
	if err != nil {
		return nil, err
	}

	evs := make([]Reduction, length)
	for i := uint64(0); i < length; i++ {
		rev := NewReduction(header.Header{})
		if err := UnmarshalReduction(r, rev); err != nil {
			return nil, err
		}

		evs[i] = *rev
	}

	return evs, nil
}

// MarshalVoteSet marshals a slice of Reduction events to a buffer.
func MarshalVoteSet(r *bytes.Buffer, evs []Reduction) error {
	if err := encoding.WriteVarInt(r, uint64(len(evs))); err != nil {
		return err
	}

	for _, event := range evs {
		if err := MarshalReduction(r, event); err != nil {
			return err
		}
	}

	return nil
}

/********************/
/* MOCKUP FUNCTIONS */
/********************/

// MockEvent mocks a Reduction event and returns it.
// It includes a vararg iterativeIdx to help avoiding duplicates when testing
func MockReduction(hash []byte, round uint64, step uint8, keys []key.ConsensusKeys, iterativeIdx ...int) Reduction {
	idx := 0
	if len(iterativeIdx) != 0 {
		idx = iterativeIdx[0]
	}

	if idx > len(keys) {
		panic("wrong iterative index: cannot iterate more than there are keys")
	}

	hdr := header.Header{Round: round, Step: step, BlockHash: hash, PubKeyBLS: keys[idx].BLSPubKeyBytes}
	r := new(bytes.Buffer)
	_ = header.MarshalSignableVote(r, hdr)
	sigma, _ := bls.Sign(keys[idx].BLSSecretKey, keys[idx].BLSPubKey, r.Bytes())
	return Reduction{
		hdr:        hdr,
		SignedHash: sigma.Compress(),
	}
}

// MockVotes mocks a slice of Reduction events and returns it.
func MockVotes(hash []byte, round uint64, step uint8, keys []key.ConsensusKeys, amount int) []Reduction {
	var voteSet []Reduction
	for i := 0; i < amount; i++ {
		r := MockReduction(hash, round, step, keys, i)
		voteSet = append(voteSet, r)
	}

	return voteSet
}

// MockVoteSet mocks a slice of Reduction events for two adjacent steps,
// and returns it.
func MockVoteSet(hash []byte, round uint64, step uint8, keys []key.ConsensusKeys, amount int) []Reduction {
	if step < uint8(2) {
		panic("Need at least 2 steps to create an Agreement")
	}

	votes1 := MockVotes(hash, round, step-1, keys, amount)
	votes2 := MockVotes(hash, round, step, keys, amount)

	return append(votes1, votes2...)
}

// createVoteSet creates and returns a set of Reduction votes for two steps.
func createVoteSet(k1, k2 []key.ConsensusKeys, hash []byte, size int, round uint64, step uint8) (events []Reduction) {
	// We can not have duplicates in the vote set.
	duplicates := make(map[string]struct{})
	// We need 75% of the committee size worth of events to reach quorum.
	for j := 0; j < int(float64(size)*0.75); j++ {
		if _, ok := duplicates[string(k1[j].BLSPubKeyBytes)]; !ok {
			ev := MockReduction(hash, round, step-2, k1, j)
			events = append(events, ev)
			duplicates[string(k1[j].BLSPubKeyBytes)] = struct{}{}
		}
	}

	// Clear the duplicates map, since we will most likely have identical keys in each array
	for k := range duplicates {
		delete(duplicates, k)
	}

	for j := 0; j < int(float64(size)*0.75); j++ {
		if _, ok := duplicates[string(k2[j].BLSPubKeyBytes)]; !ok {
			ev := MockReduction(hash, round, step-1, k2, j)
			events = append(events, ev)
			duplicates[string(k2[j].BLSPubKeyBytes)] = struct{}{}
		}
	}

	return events
}
