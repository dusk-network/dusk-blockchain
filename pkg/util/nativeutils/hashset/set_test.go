package hashset

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOps(t *testing.T) {
	test := bytes.NewBufferString("this is a test").Bytes()
	s := New()
	s.Add(test)
	assert.True(t, s.Has(test))
	assert.False(t, s.Has(append(test, 0x1)))
	assert.Equal(t, 1, s.Size())
}

func TestSize(t *testing.T) {
	s := New()
	assert.Equal(t, 0, s.Size())
}
