// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package message

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/sortedset"
	"github.com/dusk-network/dusk-crypto/bls"
	"github.com/stretchr/testify/assert"
)

func TestStepVotes(t *testing.T) {
	set := sortedset.New()

	hash := []byte("this is a mock message")
	k1 := genKeys(&set)
	k2 := genKeys(&set)
	subset := set
	// inserting a third key in the set to better test packing and unpacking
	genKeys(&set)

	bitset := set.Bits(subset)

	s, err := bls.Sign(k1.BLSSecretKey, k1.BLSPubKey, hash)
	assert.NoError(t, err)

	s2, err := bls.Sign(k2.BLSSecretKey, k2.BLSPubKey, hash)
	assert.NoError(t, err)

	apk := bls.NewApk(k1.BLSPubKey)
	assert.NoError(t, apk.Aggregate(k2.BLSPubKey))

	s.Aggregate(s2)
	assert.NoError(t, bls.Verify(apk, hash, s))

	expectedStepVotes := NewStepVotes()
	expectedStepVotes.BitSet = bitset
	expectedStepVotes.Signature = s

	buf := new(bytes.Buffer)

	assert.NoError(t, MarshalStepVotes(buf, expectedStepVotes))

	result, err := UnmarshalStepVotes(buf)
	assert.NoError(t, err)

	assert.Equal(t, expectedStepVotes, result)
	assert.NoError(t, bls.Verify(apk, hash, result.Signature))
}

// Test that adding Reduction events to a StepVotes struct results in a properly
// aggregated public key and signature.
func TestStepVotesAdd(t *testing.T) {
	sv := NewStepVotes()
	set := sortedset.New()
	hash := []byte("this is a mock message")
	signedHash, pk := genReduction(hash, &set)
	apk := bls.NewApk(pk)

	assert.NoError(t, sv.Add(signedHash))

	signedHash, pk = genReduction(hash, &set)

	assert.NoError(t, apk.Aggregate(pk))
	assert.NoError(t, sv.Add(signedHash))

	signedHash, pk = genReduction(hash, &set)

	assert.NoError(t, apk.Aggregate(pk))
	assert.NoError(t, sv.Add(signedHash))

	assert.NoError(t, bls.Verify(apk, hash, sv.Signature))
}

func genKeys(set *sortedset.Set) key.Keys {
	k, _ := key.NewRandKeys()
	set.Insert(k.BLSPubKeyBytes)
	return k
}

func genReduction(hash []byte, set *sortedset.Set) ([]byte, *bls.PublicKey) {
	k := genKeys(set)

	s, err := bls.Sign(k.BLSSecretKey, k.BLSPubKey, hash)
	if err != nil {
		panic(err)
	}

	return s.Compress(), k.BLSPubKey
}
