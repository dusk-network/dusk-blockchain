package sam3

import "testing"

// Make sure the lookup function returns a proper result
func TestLookup(t *testing.T) {
	sam, err := NewSAM("127.0.0.1:7656")
	if err != nil {
		t.Fatal(err)
	}

	defer sam.Close()
	dest, err := sam.Lookup("zzz.i2p")
	if err != nil {
		t.Fatal(err)
	}

	if err := IsI2PAddr(dest); err != nil {
		t.Fatal(err)
	}
}
