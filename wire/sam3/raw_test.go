package sam3

import (
	"testing"
)

// Test the sending and receiving of non-repliable datagrams.
// This will open 2 SAM sessions on the I2P router and make them
// talk to each other.
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

	defer rawSession.Close()
	sam2, err := NewSAM("127.0.0.1:7656")
	if err != nil {
		t.Fatal(err)
	}

	keys2, err := sam2.NewKeys()
	if err != nil {
		t.Fatal(err)
	}

	rawSession2, err := sam2.NewRawSession("raw2", keys2, []string{}, mediumShuffle)
	if err != nil {
		t.Fatal(err)
	}

	defer rawSession2.Close()
	if _, err := rawSession2.Write([]byte("Test"), keys.Addr); err != nil {
		t.Fatal(err)
	}

	msg, _, _, err := rawSession.Read()
	if err != nil {
		t.Fatal(err)
	}

	t.Log(string(msg))
}
