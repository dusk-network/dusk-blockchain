package rangeproof

import (
	"bytes"
	"math/big"
	"math/rand"
	"testing"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProveBulletProof(t *testing.T) {

	p := generateProof(3, t)

	// Verify
	ok, err := Verify(*p)
	assert.Equal(t, nil, err)
	assert.Equal(t, true, ok)

}

func TestEncodeDecode(t *testing.T) {
	p := generateProof(4, t)
	includeCommits := false

	buf := &bytes.Buffer{}
	err := p.Encode(buf, includeCommits)
	assert.Nil(t, err)

	var decodedProof Proof
	err = decodedProof.Decode(buf, includeCommits)
	assert.Nil(t, err)

	ok := decodedProof.Equals(*p, includeCommits)
	assert.True(t, ok)
}

func TestComputeMu(t *testing.T) {
	var one ristretto.Scalar
	one.SetOne()

	var expected ristretto.Scalar
	expected.SetBigInt(big.NewInt(2))

	res := computeMu(one, one, one)

	ok := expected.Equals(&res)

	assert.Equal(t, true, ok)
}

func generateProof(m int, t *testing.T) *Proof {

	// XXX: m must be a multiple of two due to inner product proof
	amounts := []ristretto.Scalar{}

	for i := 0; i < m; i++ {

		var amount ristretto.Scalar
		n := rand.Int63()
		amount.SetBigInt(big.NewInt(n))

		amounts = append(amounts, amount)
	}

	// Prove
	p, err := Prove(amounts, true)
	require.Nil(t, err)
	return &p
}
func BenchmarkProve(b *testing.B) {

	var amount ristretto.Scalar

	amount.SetBigInt(big.NewInt(100000))

	for i := 0; i < 100; i++ {

		// Prove
		Prove([]ristretto.Scalar{amount}, false)
	}

}
func BenchmarkVerify(b *testing.B) {

	var amount ristretto.Scalar

	amount.SetBigInt(big.NewInt(100000))
	p, _ := Prove([]ristretto.Scalar{amount}, false)

	b.ResetTimer()

	for i := 0; i < 100; i++ {
		// Verify
		Verify(p)
	}

}
