package fiatshamir

import ristretto "github.com/bwesterb/go-ristretto"

// HashCacher will be used for the Fiat-Shamir
// transform and will cache the necesarry values of the transcript
type HashCacher struct {
	Cache []byte
}

// Append will add a new value to the current cache
func (h *HashCacher) Append(vals ...[]byte) {

	for _, x := range vals {
		h.Cache = append(h.Cache, x...)
	}
}

// Result will return the current byte slice in
// the cache
func (h *HashCacher) Result() []byte {
	return h.Cache
}

// Clear will clear the current cache
func (h *HashCacher) Clear() {
	h.Cache = []byte{}
}

// Derive will turn the data in the cache
// into a point on the ristretto curve
func (h *HashCacher) Derive() ristretto.Scalar {
	var s ristretto.Scalar
	s.Derive(h.Cache)
	return s
}
