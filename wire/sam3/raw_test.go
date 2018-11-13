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

	rawSession, err := sam.NewRawSession("raw3", keys, []string{}, mediumShuffle)
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

	rawSession2, err := sam2.NewRawSession("raw4", keys2, []string{}, mediumShuffle)
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
