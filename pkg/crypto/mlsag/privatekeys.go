package mlsag

import ristretto "github.com/bwesterb/go-ristretto"

type PrivKeys []ristretto.Scalar

func (p *PrivKeys) AddPrivateKey(key ristretto.Scalar) {
	*p = append(*p, key)
}

func (p PrivKeys) Len() int {
	return len(p)
}
