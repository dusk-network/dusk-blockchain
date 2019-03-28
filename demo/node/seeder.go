package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"strings"
)

const secret = "v1W0imI82rqW2hT7odpI-"

func ConnectToSeeder() []string {
	conn, err := net.Dial("tcp", "voucher.dusk.network:8081")
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	fmt.Println("connected to voucher seeder")

	if err := completeChallenge(conn); err != nil {
		panic(err)
	}
	fmt.Println("challenge completed")

	// get IP list
	buf := make([]byte, 2048)
	if _, err := conn.Read(buf); err != nil {
		panic(err)
	}

	return strings.Split(string(buf), ",")
}

func completeChallenge(conn net.Conn) error {
	// wait for voucher to send challenge
	buf := make([]byte, 64)
	if _, err := conn.Read(buf); err != nil {
		return err
	}

	// get generated string out
	generated := strings.Split(string(buf), "\n")[0]

	// hash it with the secret
	hash := sha256.New()
	if _, err := hash.Write(append([]byte(generated), []byte(secret)...)); err != nil {
		return err
	}

	// turn into uppercase string
	ret := strings.ToUpper(hex.EncodeToString(hash.Sum(nil))) + "\n"

	// write response
	if _, err := conn.Write([]byte(ret)); err != nil {
		return err
	}

	return nil
}
