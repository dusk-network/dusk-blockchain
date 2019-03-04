package zkproof

// #cgo LDFLAGS: -L./ -lblindbid -framework Security
// #include "./libblindbid.h"
import "C"
import (
	"bytes"
	"math/big"
	"math/rand"
	"time"
	"unsafe"

	ristretto "github.com/bwesterb/go-ristretto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
)

// The number of rounds or each mimc hash
const mimcRounds = 90

// constants used in MIMC
var constants = genConstants()
var conBytes = constantsToBytes(constants)

// Prove creates a zkproof using d,k, seed and pubList
// This will be accessed by the consensus
// This will return the proof as a byte slice
func Prove(d, k, seed ristretto.Scalar, pubList []ristretto.Scalar) ([]byte, []byte, []byte, []byte) {

	// generate intermediate values
	q, x, y, yInv, z := prog(d, k, seed)

	dBytes := d.Bytes()
	kBytes := k.Bytes()
	yBytes := y.Bytes()
	yInvBytes := yInv.Bytes()
	qBytes := q.Bytes()
	zBytes := z.Bytes()
	seedBytes := seed.Bytes()

	dPtr := toPtr(dBytes)
	kPtr := toPtr(kBytes)
	yPtr := toPtr(yBytes)
	yInvPtr := toPtr(yInvBytes)
	qPtr := toPtr(qBytes)
	zPtr := toPtr(zBytes)
	seedPtr := toPtr(seedBytes)

	// shuffle x in slice
	pubList, i := shuffle(x, pubList)
	index := C.uint8_t(i)

	pL := make([]byte, 0, 32*len(pubList))
	for i := 0; i < len(pubList); i++ {
		pL = append(pL, pubList[i].Bytes()...)
	}

	pubListBuff := C.struct_Buffer{
		ptr: sliceToPtr(pL),
		len: C.size_t(len(pL)),
	}

	constListBuff := C.struct_Buffer{
		ptr: sliceToPtr(conBytes),
		len: C.size_t(len(conBytes)),
	}

	result := C.prove(dPtr, kPtr, yPtr, yInvPtr, qPtr, zPtr, seedPtr, &pubListBuff, &constListBuff, index)
	data := bufferToBytes(*result)

	return data, q.Bytes(), z.Bytes(), pL
}

// Verify take a proof in byte format and returns true or false depending on whether
// it is successful
func Verify(proof, seed, pubList, q, zImg []byte) bool {
	pBuf := C.struct_Buffer{
		ptr: sliceToPtr(proof),
		len: C.size_t(len(proof)),
	}

	qPtr := toPtr(q)
	zImgPtr := toPtr(zImg)
	seedPtr := sliceToPtr(seed)

	pubListBuff := C.struct_Buffer{
		ptr: sliceToPtr(pubList),
		len: C.size_t(len(pubList)),
	}

	constListBuff := C.struct_Buffer{
		ptr: sliceToPtr(conBytes),
		len: C.size_t(len(conBytes)),
	}

	verified := C.verify(&pBuf, seedPtr, &pubListBuff, qPtr, zImgPtr, &constListBuff)

	if verified {
		return true
	}

	return false
}

// CalculateX calculates the blind bid X
func CalculateX(d, m ristretto.Scalar) user.Bid {
	x := mimcHash(d, m)
	var bid user.Bid
	copy(bid[:], x.Bytes()[:])
	return bid
}

// CalculateM calculates H(k)
func CalculateM(k ristretto.Scalar) ristretto.Scalar {
	zero := ristretto.Scalar{}
	zero.SetZero()

	m := mimcHash(k, zero)
	return m
}

//Shuffle will shuffle the x value in the slice
// returning the index of the newly shuffled item and the slice
func shuffle(x ristretto.Scalar, vals []ristretto.Scalar) ([]ristretto.Scalar, uint8) {

	var index uint8

	// append x to slice
	values := append(vals, x)
	values = removeDuplicates(x, values)

	r := rand.New(rand.NewSource(time.Now().Unix()))

	ret := make([]ristretto.Scalar, len(values))
	perm := r.Perm(len(values))
	for i, randIndex := range perm {
		ret[i] = values[randIndex]
		if ret[i].Equals(&x) {
			index = uint8(i)
		}
	}
	return ret, index
}

func removeDuplicates(x ristretto.Scalar, values []ristretto.Scalar) []ristretto.Scalar {
	foundOnce := false
	for i, v := range values {
		if !bytes.Equal(x.Bytes(), v.Bytes()) {
			continue
		}

		if !foundOnce {
			foundOnce = true
			continue
		}

		values = append(values[:i], values[i+1:]...)
	}

	return values
}

// genConstants will generate the constants for
// MIMC rounds
func genConstants() []ristretto.Scalar {
	constants := make([]ristretto.Scalar, mimcRounds)
	var seed = []byte("blind bid")
	for i := 0; i < len(constants); i++ {
		c := ristretto.Scalar{}
		c.Derive(seed)
		constants[i] = c
		seed = c.Bytes()
	}

	return constants
}

func prog(d, k, seed ristretto.Scalar) (ristretto.Scalar, ristretto.Scalar, ristretto.Scalar, ristretto.Scalar, ristretto.Scalar) {

	zero := ristretto.Scalar{}
	zero.SetZero()

	m := mimcHash(k, zero)

	x := mimcHash(d, m)

	y := mimcHash(seed, x)

	yInv := ristretto.Scalar{}
	yInv.Inverse(&y)

	z := mimcHash(seed, m)

	q := ristretto.Scalar{}
	q.Mul(&d, &yInv)

	return q, x, y, yInv, z
}

func mimcHash(left, right ristretto.Scalar) ristretto.Scalar {
	x := left
	key := right

	for i := 0; i < mimcRounds; i++ {
		a := ristretto.Scalar{}
		a2 := ristretto.Scalar{}
		a3 := ristretto.Scalar{}
		a4 := ristretto.Scalar{}

		// a = x + key + constants[i]
		a.Add(&x, &key).Add(&a, &constants[i])

		// a^2
		a2.Square(&a)

		// a ^3
		a3.Mul(&a2, &a)

		//a^4
		a4.Mul(&a3, &a)

		// a_7
		x.Mul(&a4, &a3)
	}

	x.Add(&x, &key)

	return x

}

func bufferToBytes(buf C.struct_Buffer) []byte {
	return C.GoBytes(unsafe.Pointer(buf.ptr), C.int(buf.len))
}

func constantsToBytes(cconstants []ristretto.Scalar) []byte {
	c := make([]byte, 0, 90*32)
	for i := 0; i < len(constants); i++ {
		c = append(c, constants[i].Bytes()...)
	}
	return c
}

// BytesToScalar will take a slice of bytes and return it as a scalar.
func BytesToScalar(d []byte) ristretto.Scalar {
	x := ristretto.Scalar{}

	var buf [32]byte
	copy(buf[:], d[:])
	x.SetBytes(&buf)
	return x
}

// Uint64ToScalar will turn a uint64 into a scalar and return it.
func Uint64ToScalar(n uint64) ristretto.Scalar {
	x := ristretto.Scalar{}

	x.SetBigInt(big.NewInt(0).SetUint64(n))
	return x
}
