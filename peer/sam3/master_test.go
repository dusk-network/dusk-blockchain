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

	// Make sure the master session contains all IDs
	assert.Equal(t, 3, len(master.SIDs))

	// Close session
	if err := sam.Close(); err != nil {
		t.Fatal(err)
	}

	// Make sure the master session closes all it's subsessions on Close
	assert.Empty(t, master.SIDs)
}

// Set up subsessions of each type and test their functionality
// as subsessions, as well as the capability of a master session
// to run all three simultaneously.
func TestAll(t *testing.T) {
	sam, err := NewSAM("127.0.0.1:7656")
	if err != nil {
		t.Fatal(err)
	}

	defer sam.Close()
	keys, err := sam.NewKeys()
	if err != nil {
		t.Fatal(err)
	}

	master, err := sam.NewMasterSession("master", keys, mediumShuffle)
	if err != nil {
		t.Fatal(err)
	}

	raw, err := master.AddRaw("raw", []string{})
	if err != nil {
		t.Fatal(err)
	}

	dg, err := master.AddDatagram("dg", []string{})
	if err != nil {
		t.Fatal(err)
	}

	stream, err := master.AddStream("stream", []string{})
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

	master2, err := sam2.NewMasterSession("master2", keys2, mediumShuffle)
	if err != nil {
		t.Fatal(err)
	}

	raw2, err := master2.AddRaw("raw2", []string{})
	if err != nil {
		t.Fatal(err)
	}

	dg2, err := master2.AddDatagram("dg2", []string{})
	if err != nil {
		t.Fatal(err)
	}

	stream2, err := master2.AddStream("stream2", []string{})
	if err != nil {
		t.Fatal(err)
	}

	// Streaming

	// Start accepting on stream2
	go func() {
		conn, err := stream2.Accept(false)
		if err != nil {
			t.Fatal(err)
		}

		buf := make([]byte, 4096)
		n, err := conn.Read(buf)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, "Test\n", string(buf[:n]))
		if _, err := conn.Write([]byte("Test 2\n")); err != nil {
			t.Fatal(err)
		}
	}()

	// Write to stream2
	conn, err := stream.Connect(stream2.Keys.Addr)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := conn.Write([]byte("Test\n")); err != nil {
		t.Fatal(err)
	}

	// Read response
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "Test 2\n", string(buf[:n]))

	// Non-repliable datagrams

	if _, err := raw.Write([]byte("Foo\n"), raw2.Keys.Addr); err != nil {
		t.Fatal(err)
	}

	// Read message
	msg, err := raw2.Read()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "Foo\n", string(msg))

	// Repliable datagrams

	if _, err := dg.Write([]byte("Bar\n"), dg2.Keys.Addr); err != nil {
		t.Fatal(err)
	}

	// Read message
	msg2, dest, err := dg2.Read()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "Bar\n", string(msg2))
	assert.Equal(t, dg.Keys.Addr, dest)
}
