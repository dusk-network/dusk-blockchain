package msg

import (
	"github.com/dusk-network/dusk-crypto/bls"
)

// VerifyBLSSignature returns err if signature is invalid
func VerifyBLSSignature(pubKeyBytes, message, signature []byte) error {
	pubKeyBLS := &bls.PublicKey{}
	if err := pubKeyBLS.Unmarshal(pubKeyBytes); err != nil {
		return err
	}

	sig := &bls.Signature{}
	if err := sig.Decompress(signature); err != nil {
		return err
	}

	apk := bls.NewApk(pubKeyBLS)
	return bls.Verify(apk, message, sig)
}

//func VerifyBLSMultisig(apk, message, signature []byte) error {
//	batchPK, err := bls.UnmarshalApk(apk)
//	if err != nil {
//		return err
//	}
//
//	sig := &bls.Signature{}
//	if err := sig.Decompress(signature); err != nil {
//		return err
//	}
//
//	return bls.Verify(batchPK, message, sig)
//}
