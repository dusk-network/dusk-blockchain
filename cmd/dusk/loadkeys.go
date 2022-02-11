// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	consensuskey "github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/util"
	"golang.org/x/crypto/ssh/terminal"
)

// loadConsensusKeys tries to read and decrypt Consensus keys an external file defined in consensus.keysfile config.
// if the external file does not exist, it generates a new one with user-defined password.
func loadConsensusKeys() (consensuskey.Keys, error) {
	var passwd string

	filePath := cfg.Get().Consensus.KeysFile

	if len(filePath) == 0 {
		return consensuskey.Keys{}, errors.New("file not provided")
	}

	// Generate a brand new encrypted Consensus Keys file, if file does not exist
	_, errFile := os.Stat(filePath)
	if os.IsNotExist(errFile) {
		// Keys file does not exist.
		// To generate a new pair of BLS keys and saves it to
		// cfg.Get().Consensus.KeysFile a password should be provided.
		for {
			pw, err := getPassword("Enter password:")
			if err != nil {
				log.Panic(err)
			}

			// Disallow empty passwords
			if len(pw) == 0 {
				continue
			}

			var pw2 string

			pw2, err = getPassword("Confirm password:")
			if err != nil {
				log.Panic(err)
			}

			if pw != pw2 {
				continue
			}

			// Create new keys
			passwd = pw
			keys := consensuskey.NewRandKeys()

			if err := keys.Save(passwd, filePath); err != nil {
				return consensuskey.Keys{}, err
			}

			log.WithField("pubkey", util.StringifyBytes(keys.BLSPubKey)).Info("create new consensus keys")
			break
		}
	}

	// Password is not provided on generating a new consensus keys. Then we need
	// to read it from ENV variable DUSK_CONSENSUS_KEYS_PASS or terminal
	if len(passwd) == 0 {
		var err error

		pw, envVarFound := os.LookupEnv("DUSK_CONSENSUS_KEYS_PASS")
		if !envVarFound {
			pw, err = getPassword("[Open file] Enter password:")
			if err != nil {
				log.Panic(err)
			}
		}

		passwd = pw
	}

	// Try to load Consensus Keys
	keys, err := consensuskey.NewFromFile(passwd, filePath)
	if err != nil {
		return consensuskey.Keys{}, err
	}

	log.WithField("pubkey", util.StringifyBytes(keys.BLSPubKey)).Info("load consensus keys")

	return *keys, nil
}

func getPassword(prompt string) (string, error) {
	pw, err := readPassword(prompt)
	return string(pw), err
}

// This is to bypass issue with stdin from non-tty.
func readPassword(prompt string) ([]byte, error) {
	fd := int(os.Stdin.Fd())
	if terminal.IsTerminal(fd) {
		fmt.Fprintln(os.Stderr, prompt)
		return terminal.ReadPassword(fd)
	}

	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return scanner.Bytes(), nil
	}

	return nil, scanner.Err()
}
