// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package responding

import (
	"bytes"
	"crypto/sha256"
	"os"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
)

// CompleteChallenge given by a voucher seeder.
func CompleteChallenge(srcPeerID string, m message.Message) ([]bytes.Buffer, error) {
	challenge := m.Payload().(*message.Challenge)

	hash := sha256.New()
	if _, err := hash.Write(append(challenge.ChallengeString, []byte(os.Getenv("SEEDER_KEY"))...)); err != nil {
		return nil, err
	}

	resp := &message.Response{
		HashedChallenge: hash.Sum(nil),
		Port:            cfg.Get().Network.Port,
	}

	responseBuf := new(bytes.Buffer)
	if err := resp.Encode(responseBuf); err != nil {
		return nil, err
	}

	if err := topics.Prepend(responseBuf, topics.Response); err != nil {
		return nil, err
	}

	// In addition to the challenge response, let's immediately ask the voucher
	// seeder for some new addresses.
	getAddrBuf := new(bytes.Buffer)
	if err := topics.Prepend(getAddrBuf, topics.GetAddrs); err != nil {
		return nil, err
	}

	return []bytes.Buffer{*responseBuf, *getAddrBuf}, nil
}
