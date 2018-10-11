package transactions

type Stealth struct {
	Version uint8          // 1 byte
	Type    uint8          // 1 byte
	R       []byte         // 32 bytes
	TA      TypeAttributes // (m * 2565) + 32 + (n * 40) m = # inputs, n = # of outputs
}

type TypeAttributes struct {
	Inputs   []Input  //  m * 2565 bytes
	TxPubKey []byte   // 32 bytes
	Outputs  []Output // n * 40 bytes
}

type Input struct {
	KeyImage  []byte // 32 bytes
	TxID      []byte // 32 bytes
	Index     uint8  // 1 byte
	Signature []byte // ~2500 bytes
}

type Output struct {
	Amount uint64 // 8 bytes
	P      []byte // 32 bytes
}
