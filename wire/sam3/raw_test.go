package sam3

import (
	"testing"
)

func TestRaw(t *testing.T) {
	sam, err := NewSAM("127.0.0.1:7656")
	if err != nil {
		t.Fatal(err)
	}

	keys, err := sam.NewKeys()
	if err != nil {
		t.Fatal(err)
	}

	rawSession, err := sam.NewRawSession("raw", keys, []string{}, mediumShuffle)
	if err != nil {
		t.Fatal(err)
	}

	rawSession.Close()
}
