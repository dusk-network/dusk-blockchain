// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package dupemap_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/dupemap"
	"github.com/stretchr/testify/assert"
)

var dupeFilterTests = []struct {
	data   uint16
	canFwd bool
}{
	{1, true},
	{1, false},
	{2, true},
	{4, true},
	{4, false},
	{5, true},
	{7, true},
	{7, false},
	{7, false},
	{7, false},
	{9, true},
}

func TestHasAnywhere(t *testing.T) {
	dupeMap := dupemap.NewDupeMap(5, 100)

	for i, tt := range dupeFilterTests {
		test := make([]byte, 2)
		binary.BigEndian.PutUint16(test, tt.data)

		res := dupeMap.HasAnywhere(bytes.NewBuffer(test))
		if !assert.Equal(t, tt.canFwd, res) {
			assert.FailNowf(t, "failure", "DupeMap.HasAnywhere: expected %t, got %t, index %d", res, tt.canFwd, i)
		}
	}
}

func TestHasAnywhereBigData(t *testing.T) {
	type testu struct {
		payload *bytes.Buffer
		canFwd  bool
	}

	testData := make([]testu, 0)

	// Populate test data with N distinct values
	for i := uint32(0); i < 800*1000; i++ {
		d := make([]byte, 4)
		binary.BigEndian.PutUint32(d, i)
		payload := bytes.NewBuffer(d)
		testData = append(testData, testu{payload, false})
	}

	// Initialize a dupemap with 1M capacity per round-filter
	itemsCount := uint32(1000 * 1000)
	dupeMap := dupemap.NewDupeMap(10, itemsCount)

	falsePositiveCount := uint(0)

	for _, d := range testData {
		// underlying filter structure is a probabilistic data structure
		// That's said, Few false positive are possible.
		if !dupeMap.HasAnywhere(d.payload) {
			falsePositiveCount++
		}
	}

	// Ensure false positive rate is less than 1.0%
	falsePositiveRate := float64(100*falsePositiveCount) / float64(itemsCount)
	if falsePositiveRate > 1.0 {
		assert.Failf(t, "failure", "false positive are too many %f", falsePositiveRate)
	}

	// Now HasAnywhere should always returns false
	for _, d := range testData {
		// Ensure that the underlying filter structure supports "definitely
		// no" a.k.a no false negative
		if dupeMap.HasAnywhere(d.payload) != false {
			t.FailNow()
		}
	}

	// Ensure dupemap underlying structure does not consume more than 1MB for 1M capacity
	assert.LessOrEqual(t, dupeMap.Size(), 1024*1024)
}

func BenchmarkHasAnywhere(b *testing.B) {
	b.StopTimer()

	type testu struct {
		payload *bytes.Buffer
		canFwd  bool
	}

	testData := make([]testu, 0)

	for i := uint32(0); i < 900*1001; i++ {
		d := make([]byte, 4)
		binary.BigEndian.PutUint32(d, i)
		payload := bytes.NewBuffer(d)
		testData = append(testData, testu{payload, false})
	}

	for i := 0; i < b.N; i++ {
		b.StopTimer()

		dupeMap := dupemap.NewDupeMap(5, 1000000)

		b.StartTimer()

		// CanFwd always returns true
		for _, t := range testData {
			_ = dupeMap.HasAnywhere(t.payload)
		}

		for _, t := range testData {
			_ = dupeMap.HasAnywhere(t.payload)
		}
	}
}
