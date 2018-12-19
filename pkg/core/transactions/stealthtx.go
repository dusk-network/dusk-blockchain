package transactions

// Stealth defines a stealth transaction.
type Stealth struct {
	Version uint8          // 1 byte
	Type    uint8          // 1 byte
	R       []byte         // 32 bytes
	TA      TypeAttributes // (m * 2565) + 32 + (n * 40) m = # inputs, n = # of outputs
}
