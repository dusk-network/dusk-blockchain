package util

import (
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"sort"
)

// SortHeadersByHeight sorts block headers by height.
// Sorting makes it easier to retrieve the latest height and hash.
func SortHeadersByHeight(hdrs []*payload.BlockHeader) []*payload.BlockHeader {
	sortedHeaders := hdrs
	sort.Slice(sortedHeaders,
		func(i, j int) bool {
			return sortedHeaders[i].Height < sortedHeaders[j].Height
		})
	return sortedHeaders
}
