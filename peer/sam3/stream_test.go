package sam3

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test out the Connect and Accept methods in the stream source file,
// and make sure the StreamConns are closed when Close is called.
// This will open 2 SAM sessions on the I2P router and make them talk
// to each other.
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

	// First, start accepting on stream2
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

	// Then connect on stream and write to stream2
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
	if err := sam.Close(); err != nil {
		t.Fatal(err)
	}

	if err := sam2.Close(); err != nil {
		t.Fatal(err)
	}

	assert.Empty(t, stream.Streams)
	assert.Empty(t, stream2.Streams)
}
