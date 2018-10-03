package mlsag

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/toghrulmaharramov/dusk-go/ristretto"
)

// // String conforms to the stringer interface
// func (e *Element) String() string {
// 	c := hex.EncodeToString(e.c.Bytes())
// 	r := hex.EncodeToString(e.r[0].Bytes())
// 	return "c is " + c + " r is " + r
// }

// RingSignature is the collection of signatures
type RingSignature struct {
	I       ristretto.Point    // key image
	C       ristretto.Scalar   // C val
	S       []ristretto.Scalar // S vals
	PubKeys []ristretto.Point  // PubKeys including owner
}

// Sign will create the MLSAG components that can be used to verify the owner
// Returns keyimage, a c val,

// FIXME: We are going to differ from the paper, by setting j = 0 for owner and then sorting s and pubKeys to shuffle
func Sign(m []byte, mixin []ristretto.Point) RingSignature {

	// secretKey
	var sK ristretto.Scalar
	sK.Rand()

	// pubKey pK such that pK = sK * G
	var pK ristretto.Point
	pK.ScalarMultBase(&sK)

	// secret index j
	rand.Seed(time.Now().UnixNano())
	j := rand.Intn(len(mixin))
	fmt.Println("J is ", j)

	// append owners key at random position
	pubKeys := insertElAtPosition(mixin, pK, j)

	// Hp(pK)
	var hPK ristretto.Point
	hPK.Derive(pK.Bytes())

	// I = xHp(pK)
	var I ristretto.Point
	I.ScalarMult(&hPK, &sK)

	// alpha E Zq , where q is G
	var alpha ristretto.Scalar
	alpha.Rand()

	// generate s_i where i =/= j and s_i E Zq
	sVals := make([]ristretto.Scalar, len(pubKeys))
	for i := 0; i < len(sVals); i++ {
		var s ristretto.Scalar
		s.Rand()
		sVals[i] = s
	}

	// Lj = alpha * G
	var Lj ristretto.Point
	Lj.ScalarMultBase(&alpha)

	// Rj = alpha * Hp(Pj)
	var Rj ristretto.Point
	Rj.ScalarMult(&hPK, &alpha)

	// c_j+1 = Hs(m, Lj, Rj)
	var prevC ristretto.Scalar
	var hCon []byte
	hCon = append(hCon, m...)
	hCon = append(hCon, Lj.Bytes()...)
	hCon = append(hCon, Rj.Bytes()...)
	prevC.Derive(hCon)

	var c0 ristretto.Scalar // c0 is the first c val

	// start at c = j+2 as j + 1 is already calculated
	for index := j + 2; ; index++ {

		i := index % (len(pubKeys) - 1)

		fmt.Println("i is", i, "j is", j)

		var L1, L2, L, R1, R2, R, tmpPubKey, HTmpPubKey ristretto.Point

		tmpPubKey = pubKeys[i]
		HTmpPubKey.Derive(tmpPubKey.Bytes())
		s := sVals[i]

		// XXX : we do not need to check who the owner is at this point?

		//  L = sG + cP
		L1.ScalarMultBase(&s)
		L2.ScalarMult(&tmpPubKey, &prevC)
		L.Add(&L1, &L2)

		// R = s * Hp(P) + prevC * I
		R1.ScalarMult(&HTmpPubKey, &s)
		R2.ScalarMult(&I, &prevC)
		R.Add(&R1, &R2)
		fmt.Println("Generated Cs", prevC)
		hCon = []byte{}
		hCon = append(hCon, m...)
		hCon = append(hCon, L.Bytes()...)
		hCon = append(hCon, R.Bytes()...)
		prevC.Derive(hCon)

		// save this for verification, as we need first C val
		if i == 0 {
			c0 = prevC

		}

		// this means our prevC is now equal to the value for j+1
		// and we have made a complete cycle
		if i == j+1 {
			break
		}

	}

	// calculate s_j = alpha - prevC * privKey
	var sj ristretto.Scalar
	sj.Mul(&prevC, &sK).Neg(&sj).Add(&sj, &alpha)
	sVals[j] = sj

	ringsig := RingSignature{
		I:       I,
		C:       c0,
		S:       sVals,
		PubKeys: pubKeys,
	}
	return ringsig
}

func Verify(m []byte, ringsig RingSignature) bool {
	// Two conditions are that:

	// c_n+1 = c1 in 1 i mod n XXX : is this needed?
	// For all i, c_i+1 = H(m, L_i, R_i)

	numPubKeys := len(ringsig.PubKeys)
	fmt.Println("Num keys", numPubKeys)
	currC := ringsig.C // this is first c value
	I := ringsig.I

	Ls := make([]ristretto.Point, numPubKeys)
	Rs := make([]ristretto.Point, numPubKeys)
	Cs := make([]ristretto.Scalar, numPubKeys)

	for i := 0; i < numPubKeys; i++ {

		var L1, L2, L, R1, R2, R, tmpPubKey, HTmpPubKey ristretto.Point

		// calculate L and R
		tmpPubKey = ringsig.PubKeys[i]
		HTmpPubKey.Derive(tmpPubKey.Bytes())
		s := ringsig.S[i]

		//  L = sG + cP
		L1.ScalarMultBase(&s)
		L2.ScalarMult(&tmpPubKey, &currC)
		L.Add(&L1, &L2)
		Ls[i] = L

		// R = s * Hp(P) + currC * I
		R1.ScalarMult(&HTmpPubKey, &s)
		R2.ScalarMult(&I, &currC)
		R.Add(&R1, &R2)
		Rs[i] = R

		hCon := []byte{}
		hCon = append(hCon, m...)
		hCon = append(hCon, L.Bytes()...)
		hCon = append(hCon, R.Bytes()...)
		currC.Derive(hCon)

		// We need to calc and keep all C, L, R values and

		nextItem := (i + 1) % (numPubKeys - 1)
		Cs[nextItem] = currC

	}

	for i := 0; i < len(Cs); i++ {
		fmt.Println(len(Cs), numPubKeys)
		fmt.Println("Verifying Cs", Cs[i])
	}

	return true
}

// inserts an element into the slice at index
func insertElAtPosition(slice []ristretto.Point, element ristretto.Point, index int) []ristretto.Point {

	newSlice := make([]ristretto.Point, index+1)
	copy(newSlice, slice[:index])
	newSlice[index] = element

	slice = append(newSlice, slice[index:]...)

	return slice
}
