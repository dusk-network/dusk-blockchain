package main

import (
	"fmt"

	"github.com/toghrulmaharramov/dusk-go/crypto"
	"github.com/toghrulmaharramov/dusk-go/key"
)

/*
	We need:

	- Generate public address (view + spend) (A, B)
	- Generate stealth address
*/

func StealthEx() {

	Alice, _ := randKey()
	Bob, _ := randKey()
	Eve, _ := randKey()

	AliceAddr, _ := Alice.PublicAddress()
	BobAddr, _ := Bob.PublicAddress()
	EveAddr, _ := Eve.PublicAddress()

	fmt.Println(AliceAddr)
	fmt.Println(BobAddr)
	fmt.Println(EveAddr)

	// Alice Sends Money To Bob
	StealthKey, _ := key.PubAddrToKey(BobAddr)
	P, R, _ := StealthKey.StealthAddress()

	// Bob checks the P and R in the transaction to see if it is his
	x, ok := Alice.DidReceiveTx(P, R)
	if ok {
		fmt.Println("Payment was intended for Alice, unlock funds with sk", x)
		return
	}
	x, ok = Bob.DidReceiveTx(P, R)
	if ok {
		fmt.Println("Payment was intended for Bob, unlock funds with sk", x)
		return
	}
	x, ok = Eve.DidReceiveTx(P, R)
	if ok {
		fmt.Println("Payment was intended for Eve, unlock funds with sk", x)
		return
	}
	fmt.Println("This tx as not intended for any")
}

func randKey() (*key.Key, error) {
	en, err := crypto.RandEntropy(32)
	if err != nil {
		return nil, err
	}
	k, err := key.New(en)
	if err != nil {
		return nil, err
	}
	return k, nil
}
