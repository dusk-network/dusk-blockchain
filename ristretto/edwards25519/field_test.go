package edwards25519_test

import (
	"math/big"
	"math/rand"
	"os"
	"testing"

	"github.com/bwesterb/go-ristretto/edwards25519"
)

var bi25519 big.Int
var biL big.Int

var rnd *rand.Rand

func TestFeBigIntPacking(t *testing.T) {
	var bi big.Int
	var fe edwards25519.FieldElement
	for i := 0; i < 100; i++ {
		bi.Rand(rnd, &bi25519)
		fe.SetBigInt(&bi)
		if fe.BigInt().Cmp(&bi) != 0 {
			t.Fatalf("BigInt o SetBigInt != id (%v != %v)", &bi, fe.BigInt())
		}
	}
}

func TestFePacking(t *testing.T) {
	var bi big.Int
	var fe1, fe2 edwards25519.FieldElement
	for i := 0; i < 100; i++ {
		bi.Rand(rnd, &bi25519)
		fe1.SetBigInt(&bi)
		buf := fe1.Bytes()
		fe2.SetBytes(&buf)
		if !fe1.Equals(&fe2) {
			t.Fatalf("SetBytes o Bytes != id (%v != %v)", &fe1, &fe2)
		}
	}
}

// TODO test unnormalized field elements
func TestFeMul(t *testing.T) {
	var bi1, bi2, bi3 big.Int
	var fe1, fe2, fe3 edwards25519.FieldElement
	for i := 0; i < 100; i++ {
		bi1.Rand(rnd, &bi25519)
		bi2.Rand(rnd, &bi25519)
		bi3.Mul(&bi1, &bi2)
		bi3.Mod(&bi3, &bi25519)
		fe1.SetBigInt(&bi1)
		fe2.SetBigInt(&bi2)
		if fe3.Mul(&fe1, &fe2).BigInt().Cmp(&bi3) != 0 {
			t.Fatalf("%v * %v = %v != %v", &bi1, &bi2, &bi3, &fe3)
		}
	}
}

func TestFeSquare(t *testing.T) {
	var bi1, bi2 big.Int
	var fe1, fe2 edwards25519.FieldElement
	for i := 0; i < 100; i++ {
		bi1.Rand(rnd, &bi25519)
		bi2.Mul(&bi1, &bi1)
		bi2.Mod(&bi2, &bi25519)
		fe1.SetBigInt(&bi1)
		if fe2.Square(&fe1).BigInt().Cmp(&bi2) != 0 {
			t.Fatalf("%v^2 = %v != %v", &bi1, &bi2, &fe2)
		}
	}
}

func TestFeSub(t *testing.T) {
	var bi1, bi2, bi3 big.Int
	var fe1, fe2, fe3 edwards25519.FieldElement
	for i := 0; i < 100; i++ {
		bi1.Rand(rnd, &bi25519)
		bi2.Rand(rnd, &bi25519)
		bi3.Sub(&bi1, &bi2)
		bi3.Mod(&bi3, &bi25519)
		fe1.SetBigInt(&bi1)
		fe2.SetBigInt(&bi2)
		if fe3.Sub(&fe1, &fe2).BigInt().Cmp(&bi3) != 0 {
			t.Fatalf("%v - %v = %v != %v", &bi1, &bi2, &bi3, &fe3)
		}
	}
}

func TestFeAdd(t *testing.T) {
	var bi1, bi2, bi3 big.Int
	var fe1, fe2, fe3 edwards25519.FieldElement
	for i := 0; i < 100; i++ {
		bi1.Rand(rnd, &bi25519)
		bi2.Rand(rnd, &bi25519)
		bi3.Add(&bi1, &bi2)
		bi3.Mod(&bi3, &bi25519)
		fe1.SetBigInt(&bi1)
		fe2.SetBigInt(&bi2)
		if fe3.Add(&fe1, &fe2).BigInt().Cmp(&bi3) != 0 {
			t.Fatalf("%v + %v = %v != %v", &bi1, &bi2, &bi3, &fe3)
		}
	}
}

func TestFeInverse(t *testing.T) {
	var bi1, bi2 big.Int
	var fe1, fe2 edwards25519.FieldElement
	for i := 0; i < 100; i++ {
		bi1.Rand(rnd, &bi25519)
		bi2.ModInverse(&bi1, &bi25519)
		fe1.SetBigInt(&bi1)
		if fe2.Inverse(&fe1).BigInt().Cmp(&bi2) != 0 {
			t.Fatalf("1/%v = %v != %v", &bi1, &bi2, &fe2)
		}
	}
}

func TestFeInvSqrtI(t *testing.T) {
	var bi big.Int
	var fe1, fe2, feI edwards25519.FieldElement
	feI.SetI()
	for i := 0; i < 100; i++ {
		bi.Rand(rnd, &bi25519)
		fe1.SetBigInt(&bi)
		sq := fe2.InvSqrtI(&fe1)
		if sq == 0 {
			fe1.Mul(&fe1, &feI)
		}
		fe2.Mul(&fe2, &fe1)
		fe2.Square(&fe2)
		if !fe1.Equals(&fe2) {
			t.Fatalf("InvSqrtI(%v) incorrect", &bi)
		}
	}
}

func BenchmarkFeInverse(b *testing.B) {
	var fe edwards25519.FieldElement
	var bi big.Int
	bi.Rand(rnd, &bi25519)
	fe.SetBigInt(&bi)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		fe.Inverse(&fe)
	}
}

func BenchmarkFeSquare(b *testing.B) {
	var fe edwards25519.FieldElement
	var bi big.Int
	bi.Rand(rnd, &bi25519)
	fe.SetBigInt(&bi)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		fe.Square(&fe)
	}
}

func BenchmarkFeMul(b *testing.B) {
	var fe edwards25519.FieldElement
	var bi big.Int
	bi.Rand(rnd, &bi25519)
	fe.SetBigInt(&bi)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		fe.Mul(&fe, &fe)
	}
}

func BenchmarkFeIsNonZero(b *testing.B) {
	var fe edwards25519.FieldElement
	var bi big.Int
	bi.Rand(rnd, &bi25519)
	fe.SetBigInt(&bi)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		fe.IsNonZeroI()
	}
}

func BenchmarkFePack(b *testing.B) {
	var fe edwards25519.FieldElement
	var buf [32]byte
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		fe.BytesInto(&buf)
	}
}

func BenchmarkFeUnpack(b *testing.B) {
	var fe edwards25519.FieldElement
	var buf [32]byte
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		fe.SetBytes(&buf)
	}
}

func BenchmarkFeInvSqrtI(b *testing.B) {
	var fe edwards25519.FieldElement
	var bi big.Int
	bi.Rand(rnd, &bi25519)
	fe.SetBigInt(&bi)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		fe.InvSqrtI(&fe)
	}
}

func TestMain(m *testing.M) {
	bi25519.SetString(
		"7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffed", 16)
	biL.SetString(
		"1000000000000000000000000000000014def9dea2f79cd65812631a5cf5d3ed", 16)
	rnd = rand.New(rand.NewSource(37))
	os.Exit(m.Run())
}
