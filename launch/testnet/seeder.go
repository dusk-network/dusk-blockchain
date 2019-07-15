package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	cfg "gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
)

// ConnectToSeeder initializes the connection with the Voucher Seeder
func ConnectToSeeder() []string {
	if cfg.Get().General.Network == "testnet" {
		fixedNetwork := cfg.Get().Network.Seeder.Fixed
		if len(fixedNetwork) > 0 {
			log.Infof("Fixed-network config activated")
			return fixedNetwork
		}
	}

	seeders := cfg.Get().Network.Seeder.Addresses
	if len(seeders) == 0 {
		log.Errorf("Empty list of seeder addresses")
		return nil
	}

	conn, err := net.Dial("tcp", seeders[0])
	if err != nil {
		panic(err)
	}
	log.WithField("prefix", "main").Debugln("connected to voucher seeder")

	if err := completeChallenge(conn); err != nil {
		panic(err)
	}
	log.WithField("prefix", "main").Debugln("voucher seeder challenge completed")

	// get IP list
	buf := make([]byte, 2048)
	if _, err := conn.Read(buf); err != nil {
		// panic(err) <- if the seeder return error == EOF,  return nil, dont panic
		log.WithFields(log.Fields{
			"process": "main",
			"error":   err,
		}).Errorln("error reading IPs from voucher seeder")
		return nil
	}

	// start a goroutine with a ping loop for the seeder, so it knows when we shut down
	go func() {
		for {
			time.Sleep(4 * time.Second)
			_, err := conn.Write([]byte{1})
			if err != nil {
				log.WithFields(log.Fields{
					"process": "main",
					"error":   err,
				}).Warnln("error pinging voucher seeder")
				return
			}
		}
	}()

	// Trim all trailing empty bytes
	trimmedBuf := bytes.Trim(buf, "\x00")
	return strings.Split(string(trimmedBuf), ",")
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
	ret := strings.ToUpper(hex.EncodeToString(hash.Sum(nil))) + "," + cfg.Get().Network.Port + "\n"

	// write response
	if _, err := conn.Write([]byte(ret)); err != nil {
		return err
	}

	return nil
}
