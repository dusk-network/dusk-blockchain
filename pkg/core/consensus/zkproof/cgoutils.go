package zkproof

import "unsafe"
import "C"

// utility methods for cgo

func toPtr(data []byte) *C.uchar {
	return (*C.uchar)(unsafe.Pointer(&data[0]))
}

func sliceToPtr(data []byte) *C.uchar {
	cData := C.CBytes(data)
	cDataPtr := (*C.uchar)(cData)
	return cDataPtr
}
