package main

import (
	"crypto/sha256"
	"encoding/hex"
	"net"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

func ConnectToSeeder() []string {
	conn, err := net.Dial("tcp", *voucher)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	log.WithField("prefix", "main").Debugln("connected to voucher seeder")

	if err := completeChallenge(conn); err != nil {
		panic(err)
	}
	log.WithField("prefix", "main").Debugln("voucher seeder challenge completed")

	// get IP list
	buf := make([]byte, 2048)
	if _, err := conn.Read(buf); err != nil {
		// panic(err) <- if the seeder return error == EOF,  return nil, dont panic
		return nil
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
	if _, err := hash.Write(append([]byte(generated), []byte(os.Getenv("SEEDER_KEY"))...)); err != nil {
		return err
	}

	// turn into uppercase string, add port
	ret := strings.ToUpper(hex.EncodeToString(hash.Sum(nil))) + "," + *port + "\n"

	// write response
	if _, err := conn.Write([]byte(ret)); err != nil {
		return err
	}

	return nil
}
