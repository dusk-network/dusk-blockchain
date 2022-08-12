// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package agreement

import (
	"bytes"
	"testing"

	"github.com/dusk-network/bls12_381-sign/go/cgo/bls"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/sortedset"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/assert"
)

func TestAccumulatorProcessing(t *testing.T) {
	nr := 10
	hlp := NewHelper(nr)
	hash, _ := crypto.RandEntropy(32)
	handler := NewHandler(hlp.Keys, *hlp.P, []byte{0, 0, 0, 0})
	accumulator := newAccumulator(handler, 4)

	evs := hlp.Spawn(hash)
	for _, msg := range evs {
		accumulator.Process(msg)
	}

	accumulatedAggros := <-accumulator.CollectedVotesChan
	assert.Equal(t, 7, len(accumulatedAggros))
}

func TestAccumulatorProcessingAggregation(t *testing.T) {
	nr := 10
	hlp := NewHelper(nr)
	hash, _ := crypto.RandEntropy(32)
	handler := NewHandler(hlp.Keys, *hlp.P, []byte{0, 0, 0, 0})
	accumulator := newAccumulator(handler, 4)

	evs := hlp.Spawn(hash)
	for _, msg := range evs {
		accumulator.Process(msg)
	}

	accumulatedAggros := <-accumulator.CollectedVotesChan
	assert.Equal(t, 7, len(accumulatedAggros))

	var err error

	handler = NewHandler(hlp.Keys, *hlp.P, []byte{0, 0, 0, 0})
	sigs := [][]byte{}
	comm := handler.Committee(evs[0].State().Round, evs[0].State().Step)
	pubs := new(sortedset.Set)

	for _, a := range accumulatedAggros {
		sigs = append(sigs, a.Signature())
		weight := handler.VotesFor(a.State().PubKeyBLS, evs[0].State().Round, evs[0].State().Step)

		for i := 0; i < weight; i++ {
			pubs.Insert(a.State().PubKeyBLS)
		}
	}

	sig := sigs[0]
	if len(sigs) > 1 {
		sig, err = bls.AggregateSig(sigs[0], sigs[1:]...)
		assert.Nil(t, err, "could not aggregate signatures")
	}

	// TODO: #1192 - Propagate AggrAgreement
	// Create new message
	bits := comm.Bits(*pubs)
	aggro := message.NewAggrAgreement(evs[0], bits, sig)
	println(aggro.String())
	hdr := aggro.State()

	lg.
		WithField("round", hdr.Round).
		WithField("step", hdr.Step).
		// WithField("hash", util.StringifyBytes(hdr.BlockHash)).
		WithField("BitSet", aggro.Bitset).
		Infoln("processing aggragreement")

	// Verify certificate
	comm = handler.Committee(hdr.Round, hdr.Step)

	msg := message.New(topics.AggrAgreement, aggro)
	buf, err := message.Marshal(msg)
	assert.Nil(t, err, "failed to marshal aggragreement")

	newmsg, err := message.Unmarshal(&buf, nil)
	assert.Nil(t, err, "failed to unmarshal aggragreement")
	// assert.Equal(t, msg, newmsg, "deserialized message is different")

	handler = NewHandler(hlp.Keys, *hlp.P, []byte{0, 0, 0, 0})

	// Verify certificate
	comm = handler.Committee(hdr.Round, hdr.Step)

	voters := comm.Intersect(aggro.Bitset)

	subcommittee := comm.IntersectCluster(aggro.Bitset)

	allVoters := subcommittee.TotalOccurrences()

	assert.GreaterOrEqual(t, allVoters, handler.Quorum(hdr.Round), "vote set too small - %v/%v", allVoters, handler.Quorum(hdr.Round))
	newapk, err := AggregatePks(&handler.Provisioners, voters)
	assert.Nil(t, err, "could not aggregate keys")

	deser := newmsg.Payload().(message.AggrAgreement)
	// Verify signature (AggrAgreement)
	bufh := new(bytes.Buffer)

	err = header.MarshalSignableVote(bufh, hdr)
	assert.Nil(t, err, "failed to marshal signable votes")

	err = bls.Verify(newapk, deser.AggrSig, bufh.Bytes())
	assert.Nil(t, err, "failed to verify aggragreement signature")
}
