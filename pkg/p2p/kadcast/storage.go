package kadcast

/// The bucket stores `k = 40` triplets and a single
/// peer has L  buckets where `L = len(ID)`.
type Bucket struct {
	length  uint8
	entries [40]Peer
}
