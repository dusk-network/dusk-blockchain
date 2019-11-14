package kadcast

import "testing"

func TestStuff(t *testing.T) {
	a := [4]byte{255,255,255,255}
	println(*getUintFromBytes(&a))
}
func TestPOW(t *testing.T) {
	a := Peer{
		ip: [4]byte{192, 169, 1, 1},
		port: 25519,
		id: [16]byte{22,22,22,22,22,22,22,22,22,22,22,22,22,22,22,22},
	}

	println(getMyNonce(&a))
}