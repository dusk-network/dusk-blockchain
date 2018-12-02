package mlsag

// This package references the MLSAG described on pg4: https://lab.getmonero.org/pubs/MRL-0005.pdf
import (
	"bytes"
	"math/rand"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/ristretto"
	"gitlab.dusk.network/dusk-core/dusk-go/util/bytesutil"
)

// RingSignature is the collection of signatures
type RingSignature struct {
	I       ristretto.Point    // key image
	C       ristretto.Scalar   // C val
	S       []ristretto.Scalar // S vals
	PubKeys []ristretto.Point  // PubKeys including owner
}

// Sign will create the MLSAG components that can be used to verify the owner
// Returns keyimage, a c val,
func Sign(m []byte, mixin []ristretto.Point, sK ristretto.Scalar) RingSignature {

	// pubKey pK such that pK = sK * G
	var pK ristretto.Point
	pK.ScalarMultBase(&sK)

	// secret j index
	rand.Seed(time.Now().UnixNano())
	j := rand.Intn(len(mixin) + 1)

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
	sVals := make([]ristretto.Scalar, len(mixin)+1)
	for i := 1; i < len(sVals); i++ {
		var s ristretto.Scalar
		s.Rand()
		sVals[i] = s
	}

	cVals := make([]ristretto.Scalar, len(mixin)+1)

	// Lj = alpha * G
	var Lj ristretto.Point
	Lj.ScalarMultBase(&alpha)

	// Rj = alpha * Hp(Pj)
	var Rj ristretto.Point
	Rj.ScalarMult(&hPK, &alpha)

	// c_j+1 = Hs(m, Lj, Rj)
	var cPlus1 ristretto.Scalar
	buf := new(bytes.Buffer)
	bytesutil.Append(buf, m, Lj.Bytes(), Rj.Bytes())
	cPlus1.Derive(buf.Bytes())

	jPlus1 := (j + 1) % len(cVals)
	cVals[jPlus1] = cPlus1

	pubKeys := make([]ristretto.Point, len(mixin)+1)
	pubKeys[j] = pK // add signer

	for i := j + 1; ; i++ {

		l := i % (len(pubKeys))
		if l == j {
			break
		}

		k := (i + 1) % (len(pubKeys))

		c, _, _ := computeCLR(m, I, pubKeys[l], sVals[l], cPlus1)

		cVals[k] = c
		cPlus1 = c
	}

	// calculate s_j = alpha - cPlus1 * privKey
	var sj ristretto.Scalar
	sj.Mul(&cPlus1, &sK).Neg(&sj).Add(&sj, &alpha)
	sVals[j] = sj

	ringsig := RingSignature{
		I:       I,
		C:       cVals[0],
		S:       sVals,
		PubKeys: pubKeys,
	}
	return ringsig
}

func Verify(m []byte, ringsig RingSignature) bool {

	// Two conditions are that:
	// c_n+1 = c1 in 1 i mod n
	// For all i, c_i+1 = H(m, L_i, R_i)

	numPubKeys := len(ringsig.PubKeys)

	currC := ringsig.C // this is first c value c[0]

	Ls := make([]ristretto.Point, numPubKeys)
	Rs := make([]ristretto.Point, numPubKeys)
	Cs := make([]ristretto.Scalar, numPubKeys)

	for i := 0; i < numPubKeys; i++ {

		// First calculate Cs, Ls, Rs

		cPlus1, L, R := computeCLR(m, ringsig.I, ringsig.PubKeys[i], ringsig.S[i], currC)

		k := (i + 1) % numPubKeys

		Cs[k] = cPlus1

		Ls[i] = L
		Rs[i] = R

		currC = cPlus1
	}

	// check that c_i+1 = h(m || L_i || R_i)

	if len(Ls) != len(Rs) && len(Rs) != len(Cs) {
		return false
	}

	for i := range Cs {

		buf := new(bytes.Buffer)
		bytesutil.Append(buf, m, Ls[i].Bytes(), Rs[i].Bytes())

		var cPlus1 ristretto.Scalar
		cPlus1.Derive(buf.Bytes())

		k := (i + 1) % len(Cs)

		if !Cs[k].Equals(&cPlus1) {
			return false
		}

	}

	// c_n+1 = c[0]
	if Cs[0] != ringsig.C {
		return false
	}

	return true
}

// returns C, L, R
func computeCLR(message []byte, I ristretto.Point, pubKey ristretto.Point, s ristretto.Scalar, c ristretto.Scalar) (ristretto.Scalar, ristretto.Point, ristretto.Point) {
	var L1, L2, L, R1, R2, R, tmpPubKey, HTmpPubKey ristretto.Point

	var cPlus1 ristretto.Scalar

	tmpPubKey = pubKey

	HTmpPubKey.Derive(tmpPubKey.Bytes())

	//  L_j = s_j*G + c_j*P_j
	L1.ScalarMultBase(&s)
	L2.ScalarMult(&tmpPubKey, &c)
	L.Add(&L1, &L2)

	// R_j = s_j * Hp(P_j) + c_j * I
	R1.ScalarMult(&HTmpPubKey, &s)
	R2.ScalarMult(&I, &c)
	R.Add(&R1, &R2)

	buf := new(bytes.Buffer)
	bytesutil.Append(buf, message, L.Bytes(), R.Bytes())
	cPlus1.Derive(buf.Bytes())

	return cPlus1, L, R
}
