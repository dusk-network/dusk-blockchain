package ring

import "bytes"

type Cache struct {
	cache [][]byte
}

func NewCache(cap int) *Cache {
	return &Cache{
		cache: make([][]byte, cap),
	}
}

func (c *Cache) Put(item []byte, pos uint32) {
	c.cache[pos] = item
}

func (c *Cache) Has(item []byte, pos uint32) bool {
	return bytes.Equal(item, c.cache[pos])
}
