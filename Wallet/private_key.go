import (
	"crypto/rand"
	"fmt"
)

func GenerateRandomKey(n int) ([]byte) {

	b := make([]byte, n)
	_, err := rand.Read(b)

	if err != nil {
		fmt.Println("error:", err)
		return b
	}

	return b
}

func GenerateChecksum(b []byte) ([]byte) {

	checksum := doubleSHA(b)[0:4]
	
	return checksum
}

func GeneratePrivateKey() {
	length := 32
	
	random := GenerateRandomKey(n)
	checksum := GenerateChecksum(random)
	
	result := append(random, checksum...))
	
	return result
}
