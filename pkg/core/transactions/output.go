package transactions

// Output defines an output in a stealth transaction.
type Output struct {
	Amount uint64 // 8 bytes
	P      []byte // 32 bytes
}
