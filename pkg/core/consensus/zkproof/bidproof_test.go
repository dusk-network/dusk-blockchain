package zkproof_test

import (
	"testing"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/zkproof"
)

func TestProveVerify(t *testing.T) {
	//for n := 0; n < 20; n++ {
	d := genRandScalar()
	k := genRandScalar()
	seed := genRandScalar()

	// public list of bids
	pubList := make([]ristretto.Scalar, 0, 5)
	for i := 0; i < 5; i++ {
		pubList = append(pubList, genRandScalar())
	}

	proof := zkproof.Prove(d, k, seed, pubList)
	res := zkproof.Verify(proof.Proof, seed.Bytes(), proof.ProofBidList, proof.Score, proof.Z)
	assert.Equal(t, true, res)
	//}
}

func BenchmarkProveVerify(b *testing.B) {

	b.ReportAllocs()

	d := genRandScalar()
	k := genRandScalar()
	seed := genRandScalar()

	// public list of bids
	pubList := make([]ristretto.Scalar, 0, 5)
	for i := 0; i < 5; i++ {
		pubList = append(pubList, genRandScalar())
	}
	b.ResetTimer()
	b.N = 20
	for n := 0; n < b.N; n++ {
		proof := zkproof.Prove(d, k, seed, pubList)
		zkproof.Verify(proof.Proof, seed.Bytes(), proof.ProofBidList, proof.Score, proof.Z)
	}
}

func genRandScalar() ristretto.Scalar {
	c := ristretto.Scalar{}
	c.Rand()
	return c
}

func bytesToScalar(d []byte) ristretto.Scalar {
	x := ristretto.Scalar{}

	var buf [32]byte
	copy(buf[:], d[:])
	x.SetBytes(&buf)
	return x
}
