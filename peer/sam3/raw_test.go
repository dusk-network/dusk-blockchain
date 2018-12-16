package sam3

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test the sending and receiving of non-repliable datagrams.
// This will open 2 SAM sessions on the I2P router and make them
// talk to each other.
func TestRaw(t *testing.T) {
	sam, err := NewSAM("127.0.0.1:7656")
	if err != nil {
		t.Fatal(err)
	}

	defer sam.Close()
	keys, err := sam.NewKeys()
	if err != nil {
		t.Fatal(err)
	}

	rawSession, err := sam.NewRawSession("raw16", keys, []string{}, mediumShuffle)
	if err != nil {
		t.Fatal(err)
	}

	sam2, err := NewSAM("127.0.0.1:7656")
	if err != nil {
		t.Fatal(err)
	}

	defer sam2.Close()
	keys2, err := sam2.NewKeys()
	if err != nil {
		t.Fatal(err)
	}

	rawSession2, err := sam2.NewRawSession("raw17", keys2, []string{}, mediumShuffle)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := rawSession2.Write([]byte("Test\n"), rawSession.Keys.Addr); err != nil {
		t.Fatal(err)
	}

	msg, err := rawSession.Read()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "Test\n", string(msg))
}
