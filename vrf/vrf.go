// Package vrf implements a verifiable random function using the Ristretto form
// of Edwards25519, SHA3 and the Elligator2 map.

package vrf

import (
	"bytes"
	"gitlab.dusk.network/dusk-core/dusk-go/ristretto"
	"golang.org/x/crypto/sha3"
)

const (
	PublicKeySize    = 32
	SecretKeySize    = 64
	Size             = 32
	intermediateSize = 32
	ProofSize        = 32 + 32 + intermediateSize
)

// GenerateKey creates a public/secret key pair.
func GenerateKey() (*[PublicKeySize]byte, *[SecretKeySize]byte) {
	var secretKey ristretto.Scalar
	var pk = new([PublicKeySize]byte)
	var sk = new([SecretKeySize]byte)
	var digest [64]byte

	secretKey.Rand() // Generate a new secret key
	copy(sk[:32], secretKey.Bytes())

	h := sha3.NewShake256()
	h.Write(sk[:32])
	h.Read(digest[:])

	digest[0] &= 248
	digest[31] &= 127
	digest[31] |= 64

	var A ristretto.Point
	var hBytes [32]byte
	copy(hBytes[:], digest[:32])

	var hBytesScalar ristretto.Scalar
	hBytesScalar.SetBytes(&hBytes)

	A.ScalarMultBase(&hBytesScalar) // compute public key
	A.BytesInto(pk)

	copy(sk[32:], pk[:])
	return pk, sk
}

func expandSecret(sk *[SecretKeySize]byte) (*[32]byte, *[32]byte) {
	var x, skhr = new([32]byte), new([32]byte)
	hash := sha3.NewShake256()
	hash.Write(sk[:32])
	hash.Read(x[:])
	hash.Read(skhr[:])
	x[0] &= 248
	x[31] &= 127
	x[31] |= 64
	return x, skhr
}

func Compute(m []byte, sk *[SecretKeySize]byte) []byte {
	var ii ristretto.Point
	var mScalar ristretto.Scalar
	var iiB, vrf [Size]byte

	x, _ := expandSecret(sk)
	p := hashToCurve(m)

	mScalar.SetBytes(x)
	ii.ScalarMult(p, &mScalar)
	ii.BytesInto(&iiB)
	hash := sha3.NewShake256()
	hash.Write(iiB[:]) // const length: Size
	hash.Write(m)

	hash.Read(vrf[:])
	return vrf[:]
}

// Prove returns the vrf value and a proof such that Verify(pk, d, vrf, proof) == true.
// The vrf value is the same as returned by Compute(d, sk).
//
// Prove_x(d) = tuple(c=h(d, g^r, H(d)^r), t=r-c*x, ii=H(d)^x) where r = h(x, d) is used as a source of randomness
// x = secret key, d = data, c = data with randomness, r = randomness, t = delta of randomness and data w/randomness,
func Prove(d []byte, sk *[SecretKeySize]byte) ([]byte, []byte) { // Return vrf, proof
	var cH, rH [SecretKeySize]byte // cH = hash of data with randomness
	// rH = hash of randomness
	var c, r, t ristretto.Scalar
	var ii, gr, hr ristretto.Point // ii = data, gr and hr = randomness
	var grB, hrB, iiB [Size]byte

	// Two separate 32 byte hashes from the 64 byte secret key
	x, skhr := expandSecret(sk)
	// Curve point of data
	dP := hashToCurve(d)

	var xSc ristretto.Scalar
	xSc.SetBytes(x)
	ii.ScalarMult(dP, &xSc) // ii=H(d)^x) where d = data and x = secret key
	ii.BytesInto(&iiB)      // ii = encrypted data

	// Create randomness
	// r = h(x, d) is used as a source of randomness where d = data and x = secret key hash
	hash := sha3.NewShake256()
	hash.Write(skhr[:]) // 2nd part of secret key hash used for randomness
	hash.Write(sk[32:]) // public key
	hash.Write(d)       // data
	hash.Read(rH[:])
	hash.Reset()
	r.SetReduced(&rH)

	// Encrypt randomness r
	gr.ScalarMultBase(&r)
	hr.ScalarMult(dP, &r)
	gr.BytesInto(&grB)
	hr.BytesInto(&hrB)

	hash.Write(grB[:])
	hash.Write(hrB[:])
	hash.Write(d)
	hash.Read(cH[:]) // Hash of data w/randomness
	hash.Reset()
	c.SetReduced(&cH)

	var minusC ristretto.Scalar
	minusC.Neg(&c)
	t.MulAdd(&xSc, &minusC, &r) // t=r-c*x = Delta of randomness and data w/randomness

	var proof = make([]byte, ProofSize)
	copy(proof[:32], c.Bytes())
	copy(proof[32:64], t.Bytes())
	copy(proof[64:96], iiB[:])

	// VRF_x(d) = h(d, H(d)^x)) where x = secret key and d = data
	hash.Write(iiB[:])
	hash.Write(d)
	var vrf = make([]byte, Size)
	hash.Read(vrf[:])
	return vrf, proof
}

// Verify returns true if vrf=Compute(data, sk) for the sk that corresponds to pk.
//
// Check(P, d, vrf, (c,t,ii)) = vrf == h(d, ii) && c == h(d, g^t*pkP^c, H(d)^t*ii^c)
func Verify(pkBytes, d, vrfBytes, proof []byte) bool {
	var pk, iiB, vrf, ABytes, BBytes, hCheck [Size]byte
	var scZero, cRef, c, t ristretto.Scalar

	if len(proof) != ProofSize || len(vrfBytes) != Size || len(pkBytes) != PublicKeySize {
		return false
	}
	scZero.SetZero() // Scalar zero

	copy(vrf[:], vrfBytes)
	copy(pk[:], pkBytes)
	copy(c[:32], proof[:32])   // Retrieve c = Data with randomness
	copy(t[:32], proof[32:64]) // Retrieve t = Delta of randomness and data w/randomness
	copy(iiB[:], proof[64:96]) // Retrieve ii = Data

	// First verify the vrf with vrf == h(d, ii)

	hash := sha3.NewShake256()
	hash.Write(iiB[:])
	hash.Write(d)
	hash.Read(hCheck[:]) // hCheck is supposed to be vrf
	if !bytes.Equal(hCheck[:], vrf[:]) {
		return false
	}
	hash.Reset()

	// Now verify the proof with c == h(d, g^t*pkP^c, H(d)^t*ii^c)

	var pZero ristretto.Point
	var pkP, hmtP, iicP, ii, A, B, X, Y, R, S ristretto.Point
	// Get curve point of public key, consequently checking if it is on the Curve
	if !pkP.SetBytes(&pk) {
		return false
	}

	// Get curve point of data, consequently checking if it is on the Curve
	if !ii.SetBytes(&iiB) {
		return false
	}

	X.ScalarMultBase(&t)
	Y.ScalarMult(&pkP, &c)
	A.Add(&X, &Y) // A = g^t*pkP^c
	A.BytesInto(&ABytes)

	dP := hashToCurve(d) // dP = Curve point of data
	pZero.ScalarMultBase(&scZero)
	R.ScalarMult(dP, &t)
	hmtP.Add(&pZero, &R) // hmtP = H(d)^t

	S.ScalarMult(&ii, &c)
	iicP.Add(&pZero, &S) // iicP = ii^c

	B.Add(&hmtP, &iicP) // Add hmtP and iicP to get H(d)^t*ii^c
	B.BytesInto(&BBytes)

	// Create hash = h(d, g^t*pkP^c, H(d)^t*ii^c)
	var cH [64]byte
	hash.Write(ABytes[:])
	hash.Write(BBytes[:])
	hash.Write(d)
	hash.Read(cH[:])
	cRef.SetReduced(&cH)

	return cRef.Equals(&c) // cRef must have same hash as h(d, g^t*pkP^c, H(d)^t*ii^c)
}

func hashToCurve(m []byte) *ristretto.Point {
	var p ristretto.Point
	var hmb [32]byte
	sha3.ShakeSum256(hmb[:], m)
	p.SetElligator(&hmb)
	return &p
}
