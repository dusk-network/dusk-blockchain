package peer

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/hashset"
)

func TestHas(t *testing.T) {
	testPayload := bytes.NewBufferString("This is a test")
	tmpMap := NewTmpMap(3)

	assert.False(t, tmpMap.Has(testPayload))

	s := hashset.New()
	s.Add(testPayload.Bytes())
	tmpMap.msgSets[uint64(0)] = s

	assert.True(t, tmpMap.Has(testPayload))

	tmpMap.UpdateHeight(2)
	assert.False(t, tmpMap.Has(testPayload))
	assert.False(t, tmpMap.HasAt(testPayload, 2))
	assert.True(t, tmpMap.HasAnywhere(testPayload))
	assert.True(t, tmpMap.HasAt(testPayload, uint64(0)))
}

func TestAdd(t *testing.T) {
	testPayload := bytes.NewBufferString("This is a test")
	tmpMap := NewTmpMap(3)

	assert.False(t, tmpMap.Add(testPayload))
	assert.True(t, tmpMap.Add(testPayload))

	tmpMap.UpdateHeight(4)
	assert.False(t, tmpMap.Add(testPayload))
}

func TestClean(t *testing.T) {
	testPayload := bytes.NewBufferString("This is a test")
	tmpMap := NewTmpMap(3)

	assert.False(t, tmpMap.Add(testPayload))

	tmpMap.UpdateHeight(2)
	assert.False(t, tmpMap.Add(testPayload))

	tmpMap.UpdateHeight(5)
	assert.False(t, tmpMap.Add(testPayload))

	// this should clean entries at heigth 2 and less
	tmpMap.UpdateHeight(6)
	assert.False(t, tmpMap.Add(testPayload))

	assert.True(t, tmpMap.HasAnywhere(testPayload))
	assert.True(t, tmpMap.Has(testPayload))
	assert.False(t, tmpMap.HasAt(testPayload, 2))
	assert.True(t, tmpMap.HasAt(testPayload, 5))
}
