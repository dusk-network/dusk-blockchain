package key

import ristretto "github.com/bwesterb/go-ristretto"

// StealthAddress represents a Dusk stealth adress
type StealthAddress struct {
	P ristretto.Point
}
