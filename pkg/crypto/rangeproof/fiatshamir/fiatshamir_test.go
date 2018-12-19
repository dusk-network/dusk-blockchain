package fiatshamir

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmptyHashCacher(t *testing.T) {
	hs := HashCacher{[]byte{}}
	assert.Equal(t, []byte{}, hs.Result())
}
func TestHashCacher(t *testing.T) {

	arr := []string{"hello", "world", "good", "bye"}

	hs := HashCacher{[]byte{}}

	expected := ""

	for _, word := range arr {
		hs.Append([]byte(word))
		expected += word
	}

	actual := string(hs.Result())

	assert.Equal(t, expected, actual)

	hs.Clear()

	assert.Equal(t, []byte{}, hs.Result())
}
