package ristretto_test

import (
	"encoding/hex"
	"math/big"
	"math/rand"
	"os"
	"testing"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/ristretto"
)

var biL big.Int
var rnd *rand.Rand

func TestScPacking(t *testing.T) {
	var bi big.Int
	var s, s2 ristretto.Scalar
	var buf [32]byte
	for i := 0; i < 100; i++ {
		bi.Rand(rnd, &biL)
		s.SetBigInt(&bi)
		s.BytesInto(&buf)
		s2.SetBytes(&buf)
		if s.BigInt().Cmp(s2.BigInt()) != 0 {
			t.Fatalf("Unpack o Pack != id (%v != %v)", &bi, s2.BigInt())
		}
	}
}

func TestScBigIntPacking(t *testing.T) {
	var bi big.Int
	var s ristretto.Scalar
	for i := 0; i < 100; i++ {
		bi.Rand(rnd, &biL)
		s.SetBigInt(&bi)
		if s.BigInt().Cmp(&bi) != 0 {
			t.Fatalf("BigInt o SetBigInt != id (%v != %v)", &bi, s.BigInt())
		}
	}
}

func TestScSub(t *testing.T) {
	var bi1, bi2, bi3 big.Int
	var s1, s2, s3 ristretto.Scalar
	for i := 0; i < 100; i++ {
		bi1.Rand(rnd, &biL)
		bi2.Rand(rnd, &biL)
		bi3.Sub(&bi1, &bi2)
		bi3.Mod(&bi3, &biL)
		s1.SetBigInt(&bi1)
		s2.SetBigInt(&bi2)
		if s3.Sub(&s1, &s2).BigInt().Cmp(&bi3) != 0 {
			t.Fatalf("%v - %v = %v != %v", &bi1, &bi2, &bi3, s3.BigInt())
		}
	}
}

func TestScAdd(t *testing.T) {
	var bi1, bi2, bi3 big.Int
	var s1, s2, s3 ristretto.Scalar
	for i := 0; i < 100; i++ {
		bi1.Rand(rnd, &biL)
		bi2.Rand(rnd, &biL)
		bi3.Add(&bi1, &bi2)
		bi3.Mod(&bi3, &biL)
		s1.SetBigInt(&bi1)
		s2.SetBigInt(&bi2)
		if s3.Add(&s1, &s2).BigInt().Cmp(&bi3) != 0 {
			t.Fatalf("%v + %v = %v != %v", &bi1, &bi2, &bi3, s3.BigInt())
		}
	}
}

func TestScMul(t *testing.T) {
	var bi1, bi2, bi3 big.Int
	var s1, s2, s3 ristretto.Scalar
	for i := 0; i < 100; i++ {
		bi1.Rand(rnd, &biL)
		bi2.Rand(rnd, &biL)
		bi3.Mul(&bi1, &bi2)
		bi3.Mod(&bi3, &biL)
		s1.SetBigInt(&bi1)
		s2.SetBigInt(&bi2)
		if s3.Mul(&s1, &s2).BigInt().Cmp(&bi3) != 0 {
			t.Fatalf("%v * %v = %v != %v", &bi1, &bi2, &bi3, s3.BigInt())
		}
	}
}

func TestScSquare(t *testing.T) {
	var bi1, bi2 big.Int
	var s1, s2 ristretto.Scalar
	for i := 0; i < 100; i++ {
		bi1.Rand(rnd, &biL)
		bi2.Mul(&bi1, &bi1)
		bi2.Mod(&bi2, &biL)
		s1.SetBigInt(&bi1)
		if s2.Square(&s1).BigInt().Cmp(&bi2) != 0 {
			t.Fatalf("%v^2 = %v != %v",
				&bi1, &bi2, s2.BigInt())
		}
	}
}

func TestScMulAdd(t *testing.T) {
	var bi1, bi2, bi3, bi4 big.Int
	var s1, s2, s3, s4 ristretto.Scalar
	for i := 0; i < 100; i++ {
		bi1.Rand(rnd, &biL)
		bi2.Rand(rnd, &biL)
		bi3.Rand(rnd, &biL)
		bi4.Mul(&bi1, &bi2)
		bi4.Add(&bi4, &bi3)
		bi4.Mod(&bi4, &biL)
		s1.SetBigInt(&bi1)
		s2.SetBigInt(&bi2)
		s3.SetBigInt(&bi3)
		if s4.MulAdd(&s1, &s2, &s3).BigInt().Cmp(&bi4) != 0 {
			t.Fatalf("%v * %v + %v = %v != %v",
				&bi1, &bi2, &bi3, &bi4, s4.BigInt())
		}
	}
}

func TestScInverse(t *testing.T) {
	var bi1, bi2 big.Int
	var s1 ristretto.Scalar
	for i := 0; i < 100; i++ {
		bi1.Rand(rnd, &biL)
		bi2.ModInverse(&bi1, &biL)
		s1.SetBigInt(&bi1)
		if s1.Inverse(&s1).BigInt().Cmp(&bi2) != 0 {
			t.Fatalf("1/%v = %v", &bi1, &bi2)
		}
	}
}

func TestScNeg(t *testing.T) {
	var bi1, bi2 big.Int
	var s1, s2 ristretto.Scalar
	for i := 0; i < 100; i++ {
		bi1.Rand(rnd, &biL)
		bi2.Neg(&bi1)
		bi2.Mod(&bi2, &biL)
		s1.SetBigInt(&bi1)
		if s2.Neg(&s1).BigInt().Cmp(&bi2) != 0 {
			t.Fatalf("-%v = %v != %v", &bi1, &bi2, &s2)
		}
	}
}

func TestScReduced(t *testing.T) {
	var bi1, bi2, bi512 big.Int
	var s ristretto.Scalar
	bi512.SetInt64(1).Lsh(&bi512, 512)
	for i := 0; i < 100; i++ {
		var rBuf [64]byte
		bi1.Rand(rnd, &bi512)
		bi2.Mod(&bi1, &biL)
		buf := bi1.Bytes()
		for j := 0; j < len(buf) && j < 64; j++ {
			rBuf[len(buf)-j-1] = buf[j]
		}
		s.SetReduced(&rBuf)
		if s.BigInt().Cmp(&bi2) != 0 {
			t.Fatalf("SetReduced(%v) = %v != %v %v", &bi1, &bi2, s.BigInt(), rBuf)
		}
	}

}

func TestScTextMarshaling(t *testing.T) {
	var s, s2 ristretto.Scalar
	for i := 0; i < 100; i++ {
		s.Rand()
		text, _ := s.MarshalText()
		err := s2.UnmarshalText(text)
		if err != nil {
			t.Fatalf("%v: UnmarshalText o MarshalText: %v", s, err)
		}
		if s.BigInt().Cmp(s2.BigInt()) != 0 {
			t.Fatalf("%v: UnmarshalText o MarshalText != id", s)
		}
	}
}

func TestScDerive(t *testing.T) {
	var s ristretto.Scalar
	for k, v := range map[string]string{
		"test1":     "f4f2ba0eccc056c32241b5e7f648ffe6bf870773e09104f0fd2c28fbd7fc5402",
		"ristretto": "a17454b11da0ee4f9aed08190c61781c326a0c59bb449133bacc0c75308db805",
		"decaf":     "8107e19264d3e54e9869de056c90dc245dbc097529c4a5ef0dae42e1f3cd7700",
	} {
		v2 := hex.EncodeToString(s.Derive([]byte(k)).Bytes())
		if v != v2 {
			t.Fatalf("Derive(%s) = %s != %s", k, v, v2)
		}
	}
}

func BenchmarkScDerive(b *testing.B) {
	var s ristretto.Scalar
	for n := 0; n < b.N; n++ {
		s.Derive([]byte("test"))
	}
}

func BenchmarkScRand(b *testing.B) {
	var s ristretto.Scalar
	for n := 0; n < b.N; n++ {
		s.Rand()
	}
}

func BenchmarkScMul(b *testing.B) {
	var s, t ristretto.Scalar
	for n := 0; n < b.N; n++ {
		s.Mul(&s, &t)
	}
}

func BenchmarkScSquare(b *testing.B) {
	var s ristretto.Scalar
	for n := 0; n < b.N; n++ {
		s.Square(&s)
	}
}

func BenchmarkScInverse(b *testing.B) {
	var s ristretto.Scalar
	for n := 0; n < b.N; n++ {
		s.Inverse(&s)
	}
}

func TestMain(m *testing.M) {
	biL.SetString(
		"1000000000000000000000000000000014def9dea2f79cd65812631a5cf5d3ed", 16)
	rnd = rand.New(rand.NewSource(37))
	os.Exit(m.Run())
}
