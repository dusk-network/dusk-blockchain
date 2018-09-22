package ed25519

// Both functions are placed in this package so that
// this package is standalone.

func sliceTo32Arr(s []byte) *[32]byte {
	var d [32]byte
	copy(d[:], s[:])
	return &d
}

func arrToSlice(s *[32]byte) []byte {
	return s[:]
}
