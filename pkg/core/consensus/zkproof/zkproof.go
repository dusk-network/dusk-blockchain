package zkproof

// #cgo LDFLAGS: -L./ -lblindbid -framework Security
// #include "./libblindbid.h"
import "C"
import (
	"bytes"
	"errors"
	"math/big"
	"unsafe"

	ristretto "github.com/bwesterb/go-ristretto"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
)

func Prog(ctx *user.Context) (yInv []byte, zImg []byte) {
	var ds ristretto.Scalar

	ds.SetBigInt(big.NewInt(int64(ctx.D)))
	dBytes := ds.Bytes()
	dPtr := sliceToPtr(dBytes)
	kPtr := sliceToPtr(ctx.K)
	seedPtr := sliceToPtr(ctx.Seed)

	yInv = make([]byte, 32)
	zImg = make([]byte, 32)

	xPtr := toPtr(ctx.X)
	yPtr := toPtr(ctx.Y)
	yInvPtr := toPtr(yInv)
	qPtr := toPtr(ctx.Q)
	zImgPtr := toPtr(zImg)

	C.prog(seedPtr, kPtr, dPtr, qPtr, xPtr, yPtr, yInvPtr, zImgPtr)

	return
}

// Prove generates the zero-knowledge proof of the blind-bid
func Prove(ctx *user.Context, yInv, zImg []byte) ([]byte, error) {
	pubListBuf := C.struct_Buffer{
		ptr: sliceToPtr(ctx.PubList),
		len: C.size_t(len(ctx.PubList)),
	}

	var ds ristretto.Scalar

	ds.SetBigInt(big.NewInt(int64(ctx.D)))
	dBytes := ds.Bytes()

	dPtr := sliceToPtr(dBytes)
	kPtr := sliceToPtr(ctx.K)
	seedPtr := sliceToPtr(ctx.Seed)
	yPtr := toPtr(ctx.Y)
	yInvPtr := toPtr(yInv)
	qPtr := toPtr(ctx.Q)
	zImgPtr := toPtr(zImg)

	var j uint64
	for i := 0; i < len(ctx.PubList); i += 32 {
		currX := ctx.PubList[i : i+32]
		if bytes.Equal(currX, ctx.X) {
			break
		}

		j++
	}

	result := C.prove(dPtr, kPtr, yPtr, yInvPtr, qPtr, zImgPtr, seedPtr, &pubListBuf, C.ulong(j))
	if result == nil {
		return nil, errors.New("result is nil")
	}

	return C.GoBytes(unsafe.Pointer(result.ptr), C.int(result.len)), nil
}

// Verify verifies the correctness of the zk proof
func Verify(ctx *user.Context, proof, seed, q, zImg []byte) bool {
	pubListBuf := C.struct_Buffer{
		ptr: sliceToPtr(ctx.PubList),
		len: C.size_t(len(ctx.PubList)),
	}

	proofBuf := C.struct_Buffer{
		ptr: sliceToPtr(proof),
		len: C.size_t(len(proof)),
	}

	seedPtr := sliceToPtr(seed)
	qPtr := toPtr(q)
	zImgPtr := toPtr(zImg)

	verified := C.verify(&proofBuf, seedPtr, &pubListBuf, qPtr, zImgPtr)

	return verified == C.bool(true)
}

func sliceToPtr(data []byte) *C.uchar {
	cData := C.CBytes(data)
	cDataPtr := (*C.uchar)(cData)
	return cDataPtr
}

func toPtr(data []byte) *C.uchar {
	return (*C.uchar)(unsafe.Pointer(&data[0]))
}
