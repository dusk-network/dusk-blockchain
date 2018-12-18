package fiatshamir

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmptyfiatshamir.HashCacher(t *testing.T) {
	hs := fiatshamir.HashCacher{cache: []byte{}}
	assert.Equal(t, []byte{}, hs.Result())
}
func Testfiatshamir.HashCacher(t *testing.T) {

	arr := []string{"hello", "world", "good", "bye"}

	hs := fiatshamir.HashCacher{cache: []byte{}}

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
