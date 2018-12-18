package edwards25519_test

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/bwesterb/go-ristretto/cref"
	"github.com/bwesterb/go-ristretto/edwards25519"
)

func TestElligatorAndRistretto(t *testing.T) {
	var buf, goBuf, cBuf, goBuf2, cBuf2 [32]byte
	var fe edwards25519.FieldElement
	var cp edwards25519.CompletedPoint
	var ep edwards25519.ExtendedPoint
	var ep2 edwards25519.ExtendedPoint

	var cP cref.GroupGe
	var cP2 cref.GroupGe
	var cFe cref.Fe25519

	for i := 0; i < 1000; i++ {
		rnd.Read(buf[:])

		cFe.Unpack(&buf)
		cP.Elligator(&cFe)
		cP.Pack(&cBuf)

		fe.SetBytes(&buf)
		cp.SetRistrettoElligator2(&fe)
		ep.SetCompleted(&cp)
		ep.RistrettoInto(&goBuf)

		if !bytes.Equal(cBuf[:], goBuf[:]) {
			t.Fatalf("pack o elligator ( %v ) = %v != %v", buf, cBuf, goBuf)
		}

		ep2.SetRistretto(&goBuf)
		ep2.RistrettoInto(&goBuf2)

		cP2.Unpack(&cBuf)
		cP2.Pack(&cBuf2)

		if !bytes.Equal(cBuf[:], goBuf[:]) {
			t.Fatalf("pack o unpack o pack o elligator ( %v ) = %v != %v",
				buf, cBuf2, goBuf2)
		}
	}
}

func TestPointDouble(t *testing.T) {
	var buf, cBuf, goBuf [32]byte
	var fe edwards25519.FieldElement
	var cp edwards25519.CompletedPoint
	var ep, ep2 edwards25519.ExtendedPoint

	var cFe cref.Fe25519
	var cP, cP2 cref.GroupGe

	for i := 0; i < 1000; i++ {
		rnd.Read(buf[:])

		cFe.Unpack(&buf)
		cP.Elligator(&cFe)
		cP2.Double(&cP)
		cP2.Pack(&cBuf)

		fe.SetBytes(&buf)
		cp.SetRistrettoElligator2(&fe)
		ep.SetCompleted(&cp)
		ep2.Double(&ep)
		ep2.RistrettoInto(&goBuf)

		if !bytes.Equal(cBuf[:], goBuf[:]) {
			t.Fatalf("2*%v = %v != %v", ep, ep2, cP2)
		}
	}
}

func TestPointSub(t *testing.T) {
	var buf1, buf2, cBuf, goBuf [32]byte
	var fe1, fe2 edwards25519.FieldElement
	var cp1, cp2 edwards25519.CompletedPoint
	var ep1, ep2, ep3 edwards25519.ExtendedPoint

	var cFe1, cFe2 cref.Fe25519
	var cP1, cP2, cP3 cref.GroupGe

	for i := 0; i < 1000; i++ {
		rnd.Read(buf1[:])
		rnd.Read(buf2[:])

		cFe1.Unpack(&buf1)
		cFe2.Unpack(&buf2)
		cP1.Elligator(&cFe1)
		cP2.Elligator(&cFe2)
		cP2.Neg(&cP2)
		cP3.Add(&cP1, &cP2)
		cP3.Pack(&cBuf)

		fe1.SetBytes(&buf1)
		fe2.SetBytes(&buf2)
		cp1.SetRistrettoElligator2(&fe1)
		cp2.SetRistrettoElligator2(&fe2)
		ep1.SetCompleted(&cp1)
		ep2.SetCompleted(&cp2)
		ep3.Sub(&ep1, &ep2)
		ep3.RistrettoInto(&goBuf)

		if !bytes.Equal(cBuf[:], goBuf[:]) {
			t.Fatalf("%v - %v = %v != %v", ep1, ep2, ep3, cP3)
		}
	}
}

func TestPointAdd(t *testing.T) {
	var buf1, buf2, cBuf, goBuf [32]byte
	var fe1, fe2 edwards25519.FieldElement
	var cp1, cp2 edwards25519.CompletedPoint
	var ep1, ep2, ep3 edwards25519.ExtendedPoint

	var cFe1, cFe2 cref.Fe25519
	var cP1, cP2, cP3 cref.GroupGe

	for i := 0; i < 1000; i++ {
		rnd.Read(buf1[:])
		rnd.Read(buf2[:])

		cFe1.Unpack(&buf1)
		cFe2.Unpack(&buf2)
		cP1.Elligator(&cFe1)
		cP2.Elligator(&cFe2)
		cP3.Add(&cP1, &cP2)
		cP3.Pack(&cBuf)

		fe1.SetBytes(&buf1)
		fe2.SetBytes(&buf2)
		cp1.SetRistrettoElligator2(&fe1)
		cp2.SetRistrettoElligator2(&fe2)
		ep1.SetCompleted(&cp1)
		ep2.SetCompleted(&cp2)
		ep3.Add(&ep1, &ep2)
		ep3.RistrettoInto(&goBuf)

		if !bytes.Equal(cBuf[:], goBuf[:]) {
			t.Fatalf("%v + %v = %v != %v", ep1, ep2, ep3, cP3)
		}
	}
}

func TestScalarMult(t *testing.T) {
	var buf, sBuf, cBuf, goBuf [32]byte
	var biS big.Int
	var fe edwards25519.FieldElement
	var cp edwards25519.CompletedPoint
	var ep, ep2 edwards25519.ExtendedPoint

	var cFe cref.Fe25519
	var cS cref.GroupScalar
	var cP, cP2 cref.GroupGe

	for i := 0; i < 1000; i++ {
		rnd.Read(buf[:])
		biS.Rand(rnd, &biL)
		srBuf := biS.Bytes()
		for j := 0; j < len(srBuf); j++ {
			sBuf[j] = srBuf[len(srBuf)-j-1]
		}

		cFe.Unpack(&buf)
		cS.Unpack(&sBuf)
		cP.Elligator(&cFe)
		cP2.ScalarMult(&cP, &cS)
		cP2.Pack(&cBuf)

		fe.SetBytes(&buf)
		cp.SetRistrettoElligator2(&fe)
		ep.SetCompleted(&cp)
		ep2.ScalarMult(&ep, &sBuf)
		ep2.RistrettoInto(&goBuf)

		if !bytes.Equal(cBuf[:], goBuf[:]) {
			t.Fatalf("%d: %v . %v = %v != %v", i, biS, ep, ep2, cP2)
		}
	}
}

func TestRistrettoEqualsI(t *testing.T) {
	var ep1, ep2 edwards25519.ExtendedPoint
	var torsion [4]edwards25519.ExtendedPoint
	var fe edwards25519.FieldElement
	var cp edwards25519.CompletedPoint
	var buf [32]byte
	torsion[0].SetZero()
	torsion[1].SetTorsion1()
	torsion[2].SetTorsion2()
	torsion[3].SetTorsion3()
	for i := 0; i < 1000; i++ {
		rnd.Read(buf[:])
		fe.SetBytes(&buf)
		cp.SetRistrettoElligator2(&fe)
		ep1.SetCompleted(&cp)
		for j := 0; j < 4; j++ {
			ep2.Add(&ep1, &torsion[j])
			if ep1.RistrettoEqualsI(&ep2) != 1 {
				t.Fatalf("%v + %v != %v", ep1, torsion[j], ep2)
			}
		}
	}
}

func BenchmarkElligator(b *testing.B) {
	var fe edwards25519.FieldElement
	var cp edwards25519.CompletedPoint
	var ep edwards25519.ExtendedPoint
	var buf [32]byte
	rnd.Read(buf[:])
	fe.SetBytes(&buf)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		cp.SetRistrettoElligator2(&fe)
		ep.SetCompleted(&cp)
	}
}

func BenchmarkRistrettoPack(b *testing.B) {
	var fe edwards25519.FieldElement
	var cp edwards25519.CompletedPoint
	var ep edwards25519.ExtendedPoint
	var buf [32]byte
	rnd.Read(buf[:])
	fe.SetBytes(&buf)
	cp.SetRistrettoElligator2(&fe)
	ep.SetCompleted(&cp)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		ep.RistrettoInto(&buf)
	}
}

func BenchmarkRistrettoUnpack(b *testing.B) {
	var fe edwards25519.FieldElement
	var cp edwards25519.CompletedPoint
	var ep edwards25519.ExtendedPoint
	var buf [32]byte
	rnd.Read(buf[:])
	fe.SetBytes(&buf)
	cp.SetRistrettoElligator2(&fe)
	ep.SetCompleted(&cp)
	ep.RistrettoInto(&buf)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		ep.SetRistretto(&buf)
	}
}

func BenchmarkScalarMult(b *testing.B) {
	var buf, sBuf [32]byte
	var biS big.Int
	var cp edwards25519.CompletedPoint
	var ep edwards25519.ExtendedPoint
	var fe edwards25519.FieldElement
	biS.Rand(rnd, &biL)
	srBuf := biS.Bytes()
	for j := 0; j < len(srBuf); j++ {
		sBuf[j] = srBuf[len(srBuf)-j-1]
	}
	rnd.Read(buf[:])
	fe.SetBytes(&buf)
	cp.SetRistrettoElligator2(&fe)
	ep.SetCompleted(&cp)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		ep.ScalarMult(&ep, &sBuf)
	}
}
