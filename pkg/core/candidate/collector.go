// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package candidate

import (
	"bytes"
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/committee"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/msg"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/sirupsen/logrus"
)

var (
	ErrInvalidNewBlock   = errors.New("invalid newblock message")
	ErrInvalidBlockHash  = errors.New("invalid block hash")
	ErrNotBlockGenerator = errors.New("message not signed by generator")
	ErrInvalidMsgRound   = errors.New("invalid message round")
)

type Collector struct {
	eventbus *eventbus.EventBus
	handler  *committee.Handler
	db       database.DB

	round    uint64
	step     uint8
	stepName string
}

func NewCollector(e *eventbus.EventBus, h *committee.Handler, db database.DB, round uint64) *Collector {
	return &Collector{
		eventbus: e,
		handler:  h,
		db:       db,
		round:    round,
	}
}

// UpdateStep set step/stepName
func (c *Collector) UpdateStep(step uint8, name string) {
	c.step = step
	c.stepName = name
}

// Collect put the candidate block from message.Block to DB, if message is valid.
func (c *Collector) Collect(msg message.NewBlock, msgHeader []byte) error {
	if msg.State().Round != c.round {
		return ErrInvalidMsgRound
	}

	// TODO: check msg.State().Step belongs to this Round and Iteration
	log := logrus.WithField("process", c.stepName)

	if err := c.verify(msg); err != nil {
		return ErrInvalidNewBlock
	}

	// Persist Candidate Block on disk.
	if err := c.db.Update(func(t database.Transaction) error {
		// TODO: Check a candidate from this BLSKey for this iteration has been registered
		return t.StoreCandidateMessage(msg.Candidate)
	}); err != nil {
		log.WithError(err).Errorln("could not store candidate")
	}

	// Once the event is verified, and has passed all preliminary checks,
	// we can republish it to the network.
	m := message.NewWithHeader(topics.NewBlock, msg, msgHeader)
	buf, err := message.Marshal(m)
	if err != nil {
		return err
	}

	serialized := message.NewWithHeader(m.Category(), buf, m.Header())
	_ = c.eventbus.Publish(topics.Kadcast, serialized)

	return nil
}

// verify executes a set of check points to ensure the hash of the candidate
// block has been signed by the single committee member of selection step of
// this iteration.
func (c *Collector) verify(msg message.NewBlock) error {
	if !c.handler.IsMember(msg.State().PubKeyBLS, msg.State().Round, msg.State().Step, config.ConsensusSelectionMaxCommitteeSize) {
		return ErrNotBlockGenerator
	}

	// Verify message signagure
	if err := verifySignature(msg); err != nil {
		return err
	}

	// Sanity-check the candidate block
	if err := SanityCheckCandidate(msg.Candidate); err != nil {
		return err
	}

	// Ensure candidate block hash is equal to the BlockHash of the msg.header
	hash, err := msg.Candidate.CalculateHash()
	if err != nil {
		return err
	}

	if !bytes.Equal(msg.State().BlockHash, hash) {
		return ErrInvalidBlockHash
	}

	return nil
}

func verifySignature(scr message.NewBlock) error {
	packet := new(bytes.Buffer)

	hdr := scr.State()
	if err := header.MarshalSignableVote(packet, hdr); err != nil {
		return err
	}

	return msg.VerifyBLSSignature(hdr.PubKeyBLS, scr.SignedHash, packet.Bytes())
}
