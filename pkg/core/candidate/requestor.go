// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package candidate

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/sirupsen/logrus"
	lg "github.com/sirupsen/logrus"
)

var log = lg.WithField("process", "consensus")

// Requestor serves to retrieve certain Candidate messages from peers in the
// network.
type Requestor struct {
	lock           sync.RWMutex
	requesting     bool
	publisher      eventbus.Publisher
	candidateQueue chan block.Block
}

// NewRequestor returns an initialized Requestor struct.
func NewRequestor(publisher eventbus.Publisher) *Requestor {
	return &Requestor{
		publisher:      publisher,
		candidateQueue: make(chan block.Block, 100),
	}
}

// ProcessCandidate will process a received Candidate message.
// Invalid and non-matching Candidate messages are discarded.
func (r *Requestor) ProcessCandidate(srcPeerID string, msg message.Message) ([]bytes.Buffer, error) {
	if r.isRequesting() {
		var hash []byte
		var err error

		if hash, err = Validate(msg); err != nil {
			return nil, err
		}

		logrus.WithField("hash", util.StringifyBytes(hash)).Trace("process candidate")

		cm := msg.Payload().(block.Block)
		r.candidateQueue <- cm
	}

	return nil, nil
}

// RequestCandidate will attempt to fetch a Candidate message for a given hash
// from the network.
func (r *Requestor) RequestCandidate(ctx context.Context, hash []byte) (block.Block, error) {
	r.setRequesting(true)
	defer r.setRequesting(false)

	if err := r.publishGetCandidate(hash); err != nil {
		return block.Block{}, nil
	}

	for {
		select {
		case <-ctx.Done():
			log.WithField("hash", hex.EncodeToString(hash)).Debug("failed to receive candidate from the network")
			return block.Block{}, errors.New("failed to receive candidate from the network")
		case cm := <-r.candidateQueue:
			if bytes.Equal(cm.Header.Hash, hash) {
				return cm, nil
			}
		}
	}
}

func (r *Requestor) publishGetCandidate(hash []byte) error {
	// Send a request for this specific candidate
	buf := bytes.NewBuffer(hash)
	// Ugh! Move encoding after the Gossip ffs
	if err := topics.Prepend(buf, topics.GetCandidate); err != nil {
		return err
	}

	m := message.NewWithHeader(topics.GetCandidate, *buf,
		[]byte{config.ConsensusGetCandidateNodes})

	r.publisher.Publish(topics.KadcastSendToMany, m)
	return nil
}

func (r *Requestor) setRequesting(status bool) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.requesting = status
}

func (r *Requestor) isRequesting() bool {
	r.lock.RLock()
	defer r.lock.RUnlock()

	req := r.requesting
	return req
}
