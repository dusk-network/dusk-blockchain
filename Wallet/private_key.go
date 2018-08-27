import (
	"crypto/rand"
	"fmt"
)

/*	input: int
	output: []byte
	
	Utilizes crypto/rand CSRPNG to produce a pseudo-random sequence of bytes. On Linux, Reader uses getrandom(2),
	when available, otherwise falling back to /dev/urandom. On Windows, Reader uses CryptGenRandom API.
*/

func GenerateRandomKey(n int) ([]byte) {

	b := make([]byte, n)
	_, err := rand.Read(b)

	if err != nil {
		fmt.Println("error:", err)
		return b
	}

	return b
}

/*	input: []byte
	output: []byte
	
	Utilizes doubleSHA3 function to produce truncated 2 checksum bytes of the resulting hash.
*/

func GenerateChecksum(b []byte) ([]byte) {

	checksum := doubleSHA3(b)[0:4]
	
	return checksum
}

/*	input:
	output: []byte
	
*/

func GeneratePrivateKey() {
	length := 32
	
	random := GenerateRandomKey(n)
	checksum := GenerateChecksum(random)
	
	result := append(random, checksum...))
	
	return result
}
