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
	if _, err := master.Add("DATAGRAM", "testDatagram", []string{}); err != nil {
		t.Fatal(err)
	}

	if _, err := master.Add("RAW", "testRaw", []string{}); err != nil {
		t.Fatal(err)
	}

	if _, err := master.Add("STREAM", "testStream", []string{}); err != nil {
		t.Fatal(err)
	}

	// Close master session
	master.Close()

	// Make sure the master session closes all it's subsessions on Close
	assert.Empty(t, master.SIDs)
}
