package consensus

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

//XXX: remove this test and function, once gamma func has been implemented
func TestPDF(t *testing.T) {

	Y := []byte{0, 1, 1, 1, 1, 1, 1, 1, 1, 1}

	res := GenerateScore(22000, Y)
	assert.Equal(t, uint64(18446744073709534890), res)
}
