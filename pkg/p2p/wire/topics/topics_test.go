package topics_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/agreement"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

var topicTest = []struct {
	topic topics.Topic
	tba   [topics.Size]byte
}{
	{topics.Agreement, [topics.Size]byte{98, 108, 111, 99, 107, 97, 103, 114, 101, 101, 109, 101, 110, 116, 0}},
	{topics.Reduction, [topics.Size]byte{98, 108, 111, 99, 107, 114, 101, 100, 117, 99, 116, 105, 111, 110, 0}},
	{topics.Tx, [topics.Size]byte{116, 120, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
}

var topicString = []struct {
	topic   topics.Topic
	tstring string
}{
	{topics.Agreement, "blockagreement"},
	{topics.Reduction, "blockreduction"},
	{topics.Tx, "tx"},
	{topics.Topic("pippo"), "pippo"},
}

func TestStringToTopic(t *testing.T) {
	for _, tt := range topicString {
		tpc := topics.StringToTopic(tt.tstring)
		assert.Equal(t, tt.topic, tpc)
	}
}

func TestByteArrayToTopic(t *testing.T) {
	for _, tt := range topicTest {
		tpc := topics.ByteArrayToTopic(tt.tba)
		assert.Equal(t, tt.topic, tpc)
	}
}

func TestTopicToByteArray(t *testing.T) {
	for _, tt := range topicTest {
		tpc := topics.TopicToByteArray(tt.topic)
		assert.Equal(t, tt.tba, tpc)
	}
}

func TestExtract(t *testing.T) {
	for _, tt := range topicTest {
		r := bytes.NewBuffer(tt.tba[:])
		tpc, err := topics.Extract(r)
		assert.NoError(t, err)
		assert.Equal(t, tt.topic, tpc)
	}
}

func TestPeek(t *testing.T) {

	for _, tt := range topicTest {
		r := bytes.NewBuffer(tt.tba[:])
		tpc, err := topics.Peek(r)
		assert.NoError(t, err)
		assert.Equal(t, tt.topic, tpc)
	}
}

func BenchmarkPeek(b *testing.B) {
	r := mockAggro()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = topics.Peek(r)
	}
}

func BenchmarkExtract(b *testing.B) {
	buf := mockAggro()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var newBuf bytes.Buffer
		r := io.TeeReader(buf, &newBuf)
		_, _ = topics.Extract(r)
		_, _ = ioutil.ReadAll(r)
	}
}

func mockAggro() *bytes.Buffer {
	hash, _ := crypto.RandEntropy(32)

	p, ks := consensus.MockProvisioners(3)
	aggro := agreement.MockAgreement(hash, uint64(1), uint8(2), ks, p.CreateVotingCommittee(1, 2, 3))
	_ = topics.Write(aggro, topics.Agreement)
	return aggro
}
