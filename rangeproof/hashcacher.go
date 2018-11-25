package rangeproof

import "github.com/toghrulmaharramov/dusk-go/ristretto"

// hashCacher will be used for the Fiat-Shamir
// transform and will cache the necesarry values of the transcript
type hashCacher struct {
	cache []byte
}

// Append will add a new value to the current cache
func (h *hashCacher) Append(vals ...[]byte) {

	for _, x := range vals {
		h.cache = append(h.cache, x...)
	}
}

// Result will return the current byte slice in
// the cache
func (h *hashCacher) Result() []byte {
	return h.cache
}

// Clear will clear the current cache
func (h *hashCacher) Clear() {
	h.cache = []byte{}
}

// Derive will turn the data in the cache
// into a point on the ristretto curve
func (h *hashCacher) Derive() ristretto.Scalar {
	var s ristretto.Scalar
	s.Derive(h.cache)
	return s
}
