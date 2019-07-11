package key

// PrivateKey represents the pair of private spend and view keys
type PrivateKey struct {
	privSpend *PrivateSpend
	privView  *PrivateView
}

func newPrivateKey(seed []byte) *PrivateKey {

	privSpend := newSpendFromSeed(seed)
	privView := privSpend.PrivateView()

	return &PrivateKey{
		privSpend,
		privView,
	}
}

func (prk PrivateKey) publicKey() *PublicKey {
	return &PublicKey{
		prk.privSpend.PublicSpend(),
		prk.privView.PublicView(),
	}
}
