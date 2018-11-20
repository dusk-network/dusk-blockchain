package sam3

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test the sending and receiving of repliable datagrams.
// This will open 2 SAM sessions on the I2P router and make them
// talk to each other.
func TestDatagram(t *testing.T) {
	sam, err := NewSAM("127.0.0.1:7656")
	if err != nil {
		t.Fatal(err)
	}

	defer sam.Close()
	keys, err := sam.NewKeys()
	if err != nil {
		t.Fatal(err)
	}

	dgSession, err := sam.NewDatagramSession("dg", keys, []string{}, mediumShuffle)
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

	dgSession2, err := sam2.NewDatagramSession("dg2", keys2, []string{}, mediumShuffle)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := dgSession2.Write([]byte("Test"), keys.Addr); err != nil {
		t.Fatal(err)
	}

	msg, dest, err := dgSession.Read()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "Test", string(msg))
	assert.Equal(t, dgSession2.Keys.Addr, dest)
}
