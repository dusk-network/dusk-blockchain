package transactions

type Stealth struct {
	Version uint8
	Type    uint8
	R       []byte
	TA      TypeAttributes
}

type TypeAttributes struct {
	Inputs   []Input
	TxPubKey []byte
	Outputs  []Output
}

type Input struct {
	KeyImage  []byte
	TxID      []byte
	Index     uint8
	Signature []byte
}

type Output struct {
	Amount uint64
	P      []byte
}
