// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package challenger

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"os"

	"github.com/dusk-network/dusk-blockchain/cmd/voucher/node"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	log "github.com/sirupsen/logrus"
)

const challengeLength = 20

// Challenger is the component responsible for vetting incoming connections.
type Challenger struct {
	nodes  *node.Store
	gossip *protocol.Gossip
}

// New creates a new, initialized Challenger.
func New(store *node.Store) *Challenger {
	return &Challenger{nodes: store, gossip: protocol.NewGossip()}
}

// SendChallenge to a connecting peer.
func (c *Challenger) SendChallenge(ctx context.Context, r *peer.Reader, w *peer.Writer) {
	challenge, err := generateRandomBytes(challengeLength)
	if err != nil {
		log.Panic(err)
	}

	buf := bytes.NewBuffer(challenge)
	if err := topics.Prepend(buf, topics.Challenge); err != nil {
		log.Panic(err)
	}

	if err := c.gossip.Process(buf); err != nil {
		log.WithError(err).Warnln("could not send challenge")
		return
	}

	if _, err := w.Write(buf.Bytes()); err != nil {
		log.WithError(err).Warnln("could not send challenge")
		return
	}

	// Enter the node into the store, for future reference.
	c.nodes.Add(r.Addr(), challenge)

	peer.Create(ctx, r, w)

	c.nodes.SetInactive(r.Addr())
}

// ProcessResponse will process the response to a connection challenge.
func (c *Challenger) ProcessResponse(srcPeerID string, m message.Message) ([]bytes.Buffer, error) {
	node := c.nodes.Get(srcPeerID)
	resp := m.Payload().(message.Response)

	if !hashesMatch(resp.HashedChallenge, node.Challenge) {
		c.nodes.BlackList(srcPeerID)
		return nil, errors.New("received invalid response")
	}

	// Update the node's entry with the listening port.
	c.nodes.SetPort(srcPeerID, resp.Port)
	return nil, nil
}

func generateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	return b, err
}

// Secret is a shared secret between the voucher seeder and the node.
var Secret = os.Getenv("SEEDER_KEY")

// hashChallenge will hash a string using sha256.
func hashChallenge(challenge []byte) []byte {
	sha256Local := sha256.New()
	_, _ = sha256Local.Write(challenge)
	return sha256Local.Sum(nil)
}

// HashesMatch hashes a string and compares it with a provided value.
func hashesMatch(providedHash, challenge []byte) bool {
	computedChallenge := ComputeChallenge(challenge)
	result := bytes.Equal(computedChallenge, providedHash)
	return result
}

// ComputeChallenge returns the expected answer by hashing the random string with the shared secret.
func ComputeChallenge(gen []byte) []byte {
	return hashChallenge(append(gen, []byte(Secret)...))
}
