package sam3

import (
	"testing"
	"time"
)

func TestStreamConnectAccept(t *testing.T) {
	sam, err := NewSAM("127.0.0.1:7656")
	if err != nil {
		t.Fatal(err)
	}

	keys, err := sam.NewKeys()
	if err != nil {
		t.Fatal(err)
	}

	stream, err := sam.NewStreamSession("stream", keys, []string{}, mediumShuffle)
	if err != nil {
		t.Fatal(err)
	}

	defer stream.Close()
	sam2, err := NewSAM("127.0.0.1:7656")
	if err != nil {
		t.Fatal(err)
	}

	keys2, err := sam.NewKeys()
	if err != nil {
		t.Fatal(err)
	}

	stream2, err := sam2.NewStreamSession("stream2", keys2, []string{}, mediumShuffle)
	if err != nil {
		t.Fatal(err)
	}

	defer stream2.Close()

	// First, start accepting on stream2
	go func() {
		conn, err := stream2.Accept(false)
		if err != nil {
			t.Fatal(err)
		}

		buf := make([]byte, 4096)
		if _, err := conn.Read(buf); err != nil {
			t.Fatal(err)
		}

		t.Log(string(buf))
		if _, err := conn.Write([]byte("Test 2\n")); err != nil {
			t.Fatal(err)
		}
	}()

	// Give stream2 a head start..
	time.Sleep(4 * time.Second)

	// Then connect on stream and write to stream2
	conn, err := stream.Connect(keys2.Addr)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := conn.Write([]byte("Test\n")); err != nil {
		t.Fatal(err)
	}

	// Read it
	buf := make([]byte, 4096)
	if _, err := conn.Read(buf); err != nil {
		t.Fatal(err)
	}

	t.Log(string(buf))
}
