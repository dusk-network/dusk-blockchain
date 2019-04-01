package wire

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {

	eb := New()
	assert.NotNil(t, eb)
}
