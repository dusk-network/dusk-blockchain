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
