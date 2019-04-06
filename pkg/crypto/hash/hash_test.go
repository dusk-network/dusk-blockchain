package hash

import (
	"crypto/rand"
	"testing"
)

func randomMessage(size int) []byte {
	msg := make([]byte, size)
	_, _ = rand.Read(msg)
	return msg
}

func BenchmarkXxhash(b *testing.B) {

	testBytes := randomMessage(32)

	for i := 0; i < b.N; i++ {
		_, _ = Xxhash(testBytes)
	}
}

func BenchmarkSha3(b *testing.B) {

	testBytes := randomMessage(32)

	for i := 0; i < b.N; i++ {
		_, _ = Sha3256(testBytes)
	}
}
