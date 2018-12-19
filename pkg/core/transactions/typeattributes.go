package transactions

// TypeAttributes holds the inputs, outputs and the public key associated
// with a stealth transaction.
type TypeAttributes struct {
	Inputs   []Input  //  m * 2565 bytes
	TxPubKey []byte   // 32 bytes
	Outputs  []Output // n * 40 bytes
}
