package main

import (
	"fmt"

	"gitlab.com/dusk-go/crypto/hash"
)

func h(str []byte) []byte {
	h, err := hash.Sha3256(str)
	if err != nil {
		fmt.Println(err)
		panic(err)
	} else {
		fmt.Println(h)
	}
	return h
}

func main() {
	// var toHash []byte
	// fmt.Println("Enter the word you want hashed")

	// fmt.Scanf("%s", &toHash)
	bella := []byte("Bella")
	ciao, ndo, scappi := []byte("Ciao"), []byte("Ndo"), []byte("Scappi")

	hB := h(bella)
	hC := h(ciao)
	hBC := h(append(hB, hC...))

	hN := h(ndo)
	hS := h(scappi)
	hNS := h(append(hN, hS...))

	h(append(hBC, hNS...))
}
