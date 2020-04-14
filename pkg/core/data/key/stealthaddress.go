package key

import ristretto "github.com/bwesterb/go-ristretto"

// StealthAddress represents a Dusk stealth address
type StealthAddress struct {
	P ristretto.Point
}
