package reduction

import (
	"bytes"
	"encoding/hex"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

var timerLength = 100 * time.Millisecond

func TestSimpleBlockReduction(t *testing.T) {
	keys, votingCommittee, err := mockVotingCommittee()
	if err != nil {
		t.Fatal(err)
	}

	committee := mockCommittee(1, true, nil, votingCommittee)
	hashChannel := make(chan *bytes.Buffer, 1)
	resultChannel := make(chan *bytes.Buffer, 1)
	br := newBlockReductionCollector(committee, timerLength, nil, hashChannel,
		resultChannel)
	br.updateRound(1)

	blockHash := []byte("kaasbroodje")
	unmarshaller1, err := newMockREUnmarshaller(blockHash, keys, 1, 1)
	if err != nil {
		t.Fatal(err)
	}

	br.unmarshaller = unmarshaller1
	br.Collect(bytes.NewBuffer(nil))

	// wait for something to come from the hashChannel
	result1 := <-br.hashChannel

	// first 9 bytes will contain a round and step
	assert.Equal(t, blockHash, result1.Bytes()[9:])

	unmarshaller2, err := newMockREUnmarshaller(blockHash, keys, 1, 2)
	if err != nil {
		t.Fatal(err)
	}

	br.unmarshaller = unmarshaller2
	br.Collect(bytes.NewBuffer(nil))

	result2 := <-br.resultChannel

	// first 9 bytes are round and step, everything after 41 is the voteset
	assert.Equal(t, blockHash, result2.Bytes()[9:41])
}

// create a mock voting committee with one participant
func mockVotingCommittee() (*user.Keys, map[string]uint8, error) {
	keys, err := user.NewRandKeys()
	if err != nil {
		return nil, nil, err
	}

	votingCommittee := map[string]uint8{
		hex.EncodeToString(keys.BLSPubKey.Marshal()): 1,
	}

	return keys, votingCommittee, nil
}

type mockREUnmarshaller struct {
	*Event
}

func (m *mockREUnmarshaller) Unmarshal(b *bytes.Buffer, e wire.Event) error {
	ev := e.(*BlockReduction)
	ev.Step = m.Step
	ev.Round = m.Round
	ev.VotedHash = m.VotedHash
	ev.PubKeyBLS = m.PubKeyBLS
	ev.SignedHash = m.SignedHash
	return nil
}

func newMockREUnmarshaller(blockHash []byte, keys *user.Keys, round uint64,
	step uint8) (wire.EventUnmarshaller, error) {

	ev := &BlockReduction{}
	ev.VotedHash = blockHash
	ev.Round = round
	ev.Step = step
	ev.PubKeyBLS = keys.BLSPubKey.Marshal()

	signedHash, err := bls.Sign(keys.BLSSecretKey, keys.BLSPubKey, blockHash)
	if err != nil {
		return nil, err
	}

	ev.SignedHash = signedHash.Compress()

	return &mockREUnmarshaller{
		Event: ev,
	}, nil
}
