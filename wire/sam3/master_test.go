package sam3

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test out the running of multiple sessions on a master session,
// and make sure they are closed properly on the `Close` call.
func TestSubsessions(t *testing.T) {
	// Open socket on SAM bridge
	sam, err := NewSAM("127.0.0.1:7656")
	if err != nil {
		t.Fatal(err)
	}

	// Get an address
	keys, err := sam.NewKeys()
	if err != nil {
		t.Fatal(err)
	}

	// Set up master session
	master, err := sam.NewMasterSession("master", keys, mediumShuffle)
	if err != nil {
		t.Fatal(err)
	}

	// Add subsessions
	if _, err := master.AddDatagram("dg", []string{}); err != nil {
		t.Fatal(err)
	}

	if _, err := master.AddRaw("raw", []string{}); err != nil {
		t.Fatal(err)
	}

	if _, err := master.AddStream("stream", []string{}); err != nil {
		t.Fatal(err)
	}

	// Close master session
	master.Close()

	// Make sure the master session closes all it's subsessions on Close
	assert.Empty(t, master.SIDs)
}

func TestSubsessionStreamConnectAccept(t *testing.T) {
	sam, err := NewSAM("127.0.0.1:7656")
	if err != nil {
		t.Fatal(err)
	}

	keys, err := sam.NewKeys()
	if err != nil {
		t.Fatal(err)
	}

	master, err := sam.NewMasterSession("master", keys, mediumShuffle)
	if err != nil {
		t.Fatal(err)
	}

	defer master.Close()
	stream, err := master.AddStream("stream", []string{})
	if err != nil {
		t.Fatal(err)
	}

	stream2, err := master.AddStream("stream2", []string{})
	if err != nil {
		t.Fatal(err)
	}

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

	conn, err := stream.Connect(keys.Addr)
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
