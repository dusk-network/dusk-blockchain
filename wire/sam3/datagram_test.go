package sam3

import "testing"

func TestDatagram(t *testing.T) {
	sam, err := NewSAM("127.0.0.1:7656")
	if err != nil {
		t.Fatal(err)
	}

	keys, err := sam.NewKeys()
	if err != nil {
		t.Fatal(err)
	}

	dgSession, err := sam.NewDatagramSession("dg3", keys, []string{}, mediumShuffle)
	if err != nil {
		t.Fatal(err)
	}

	defer dgSession.Close()
	sam2, err := NewSAM("127.0.0.1:7656")
	if err != nil {
		t.Fatal(err)
	}

	keys2, err := sam2.NewKeys()
	if err != nil {
		t.Fatal(err)
	}

	dgSession2, err := sam2.NewDatagramSession("dg4", keys2, []string{}, mediumShuffle)
	if err != nil {
		t.Fatal(err)
	}

	defer dgSession2.Close()
	if _, err := dgSession2.Write([]byte("Test"), keys.Addr); err != nil {
		t.Fatal(err)
	}

	msg, _, _, _, err := dgSession.Read()
	if err != nil {
		t.Fatal(err)
	}

	t.Log(string(msg))
}
