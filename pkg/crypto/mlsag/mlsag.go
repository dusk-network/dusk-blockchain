package mlsag

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	ristretto "github.com/bwesterb/go-ristretto"
)

type Signature struct {
	c       ristretto.Scalar
	r       []Responses
	PubKeys []PubKeys
	Msg     []byte
}

func (s *Signature) Encode(w io.Writer, encodeKeys bool) error {
	err := binary.Write(w, binary.BigEndian, s.c.Bytes())
	if err != nil {
		return err
	}

	// lenR is the number of response vectors == num users = num pubkey vectors
	lenR := uint32(len(s.r))
	err = binary.Write(w, binary.BigEndian, lenR)
	if err != nil {
		return err
	}

	if lenR <= 0 {
		return nil
	}

	// numResponses is the number of responses per user  == num pubkeys
	numResponses := uint32(s.r[0].Len())
	err = binary.Write(w, binary.BigEndian, numResponses)
	if err != nil {
		return err
	}

	// Encode the responses
	for i := range s.r {
		err = s.r[i].Encode(w)
		if err != nil {
			return err
		}
	}

	if !encodeKeys {
		return nil
	}

	// Encode the pubkeys
	for i := range s.PubKeys {
		err = s.PubKeys[i].Encode(w)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Signature) Decode(r io.Reader, decodeKeys bool) error {

	if s == nil {
		return errors.New("struct is nil")
	}

	err := readerToScalar(r, &s.c)
	if err != nil {
		return err
	}

	var lenR, numResponses uint32
	err = binary.Read(r, binary.BigEndian, &lenR)
	if err != nil {
		return err
	}
	err = binary.Read(r, binary.BigEndian, &numResponses)
	if err != nil {
		return err
	}

	// Decode the responses
	s.r = make([]Responses, lenR)
	for i := uint32(0); i < lenR; i++ {
		err = s.r[i].Decode(r, numResponses)
		if err != nil {
			return err
		}
	}

	if !decodeKeys {
		return nil
	}

	// Decode pubkeys
	s.PubKeys = make([]PubKeys, lenR)
	for i := uint32(0); i < lenR; i++ {
		err = s.PubKeys[i].Decode(r, numResponses)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s Signature) Equals(other Signature, includeKeys bool) bool {
	ok := s.c.Equals(&other.c)
	if !ok {
		return ok
	}

	for i := range s.r {
		ok = s.r[i].Equals(other.r[i])
		if !ok {
			return ok
		}
	}

	if !includeKeys {
		return true
	}

	if len(s.PubKeys) != len(other.PubKeys) {
		return false
	}

	for i := 0; i < len(s.PubKeys); i++ {
		ok = s.PubKeys[i].Equals(other.PubKeys[i])
		if !ok {
			return ok
		}
	}
	return true
}

func (proof *Proof) prove(skipLastKeyImage bool) (*Signature, []ristretto.Point, error) {

	proof.addSignerPubKey()

	// Shuffle the PubKeys and update the index for our corresponding key
	err := proof.shuffleSet()
	if err != nil {
		return nil, nil, err
	}

	// Check that all key vectors are the same size in pubkey matrix
	pubKeyVecLen := proof.privKeys.Len()
	for i := range proof.pubKeysMatrix {
		if proof.pubKeysMatrix[i].Len() != pubKeyVecLen {
			return nil, []ristretto.Point{}, errors.New("all vectors in the pubkey matrix must be the same size")
		}
	}

	keyImages := proof.calculateKeyImages(skipLastKeyImage)
	nonces := generateNonces(len(proof.privKeys))

	numUsers := len(proof.pubKeysMatrix)
	numKeysPerUser := len(proof.privKeys)

	// We will overwrite the signers responses
	responses := generateResponses(numUsers, numKeysPerUser, proof.index)

	// Let secretIndex = index of signer
	secretIndex := proof.index

	// Generate C_{secretIndex+1}
	buf := &bytes.Buffer{}
	buf.Write(proof.msg)
	signersPubKeys := proof.pubKeysMatrix[secretIndex]

	for i := 0; i < len(nonces); i++ {

		nonce := nonces[i]

		// P = nonce * G
		var P ristretto.Point
		P.ScalarMultBase(&nonce)
		_, err = buf.Write(P.Bytes())
		if err != nil {
			return nil, nil, err
		}
	}

	for i := 0; i < len(keyImages); i++ {

		nonce := nonces[i]

		// P = nonce * H(K)
		var P, hK ristretto.Point
		hK.Derive(signersPubKeys.keys[i].Bytes())
		P.ScalarMult(&hK, &nonce)
		_, err = buf.Write(P.Bytes())
		if err != nil {
			return nil, nil, err
		}
	}

	var CjPlusOne ristretto.Scalar
	CjPlusOne.Derive(buf.Bytes())

	// generate challenges
	challenges := make([]ristretto.Scalar, numUsers)
	challenges[(secretIndex+1)%numUsers] = CjPlusOne

	var prevChallenge ristretto.Scalar
	prevChallenge.Set(&CjPlusOne)

	for k := secretIndex + 2; k != (secretIndex+1)%numUsers; k = (k + 1) % numUsers {
		i := k % numUsers

		prevIndex := (i - 1) % numUsers
		if prevIndex < 0 {
			prevIndex = prevIndex + numUsers
		}
		fakeResponses := responses[prevIndex]
		decoyPubKeys := proof.pubKeysMatrix[prevIndex]

		c, err := generateChallenge(proof.msg, fakeResponses, keyImages, decoyPubKeys, prevChallenge)
		if err != nil {
			return nil, nil, err
		}

		challenges[i].Set(&c)
		prevChallenge.Set(&c)
	}

	// Set the real response for signer
	var realResponse Responses
	for i := 0; i < numKeysPerUser; i++ {
		challenge := challenges[proof.index]
		privKey := proof.privKeys[i]
		nonce := nonces[i]
		var r ristretto.Scalar

		// r = nonce - challenge*privKey
		r.Mul(&challenge, &privKey)
		r.Neg(&r)
		r.Add(&r, &nonce)
		realResponse.AddResponse(r)
	}

	// replace real response in responses array
	responses[proof.index] = realResponse

	sig := &Signature{
		c:       challenges[0],
		r:       responses,
		PubKeys: proof.pubKeysMatrix,
		Msg:     proof.msg,
	}

	return sig, keyImages, nil
}

func (sig *Signature) Verify(keyImages []ristretto.Point) (bool, error) {

	if len(sig.PubKeys) == 0 || len(sig.r) == 0 || len(keyImages) == 0 {
		return false, errors.New("cannot have zero length for responses, pubkeys or key images")
	}

	numUsers := len(sig.r)
	index := 0

	var prevChallenge = sig.c

	for k := index + 1; k != (index)%numUsers; k = (k + 1) % numUsers {
		i := k % numUsers
		prevIndex := (i - 1) % numUsers
		if prevIndex < 0 {
			prevIndex = prevIndex + numUsers
		}

		fakeResponses := sig.r[prevIndex]
		decoyPubKeys := sig.PubKeys[prevIndex]
		challenge, err := generateChallenge(sig.Msg, fakeResponses, keyImages, decoyPubKeys, prevChallenge)
		if err != nil {
			return false, err
		}
		prevChallenge = challenge
	}

	// Calculate c'
	prevIndex := (index - 1) % numUsers
	if prevIndex < 0 {
		prevIndex = prevIndex + numUsers
	}
	fakeResponses := sig.r[prevIndex]
	decoyPubKeys := sig.PubKeys[prevIndex]

	challenge, err := generateChallenge(sig.Msg, fakeResponses, keyImages, decoyPubKeys, prevChallenge)
	if err != nil {
		return false, err
	}

	if !challenge.Equals(&sig.c) {
		return false, fmt.Errorf("c'0 does not equal c0, %s != %s", challenge.String(), sig.c.String())
	}

	return true, nil
}

func generateNonces(n int) []ristretto.Scalar {
	var nonces []ristretto.Scalar
	for i := 0; i < n; i++ {
		var nonce ristretto.Scalar
		nonce.Rand()
		nonces = append(nonces, nonce)
	}
	return nonces
}

// XXX: Test should check that random numbers are not all zero
//A bug in ristretto lib that may not be fixed
// Check the same for points too
// skip skips the singers responses
func generateResponses(m int, n, skip int) []Responses {
	var matrixResponses []Responses
	for i := 0; i < m; i++ {
		if i == skip {
			matrixResponses = append(matrixResponses, Responses{})
			continue
		}
		var resp Responses
		for i := 0; i < n; i++ {
			var r ristretto.Scalar
			r.Rand()
			resp.AddResponse(r)
		}
		matrixResponses = append(matrixResponses, resp)
	}
	return matrixResponses
}

func generateChallenge(
	msg []byte,
	respsonses Responses,
	keyImages []ristretto.Point,
	pubKeys PubKeys,
	prevChallenge ristretto.Scalar) (ristretto.Scalar, error) {

	buf := &bytes.Buffer{}
	_, err := buf.Write(msg)
	if err != nil {
		return ristretto.Scalar{}, err
	}

	for i := 0; i < pubKeys.Len(); i++ {

		r := respsonses[i]

		// P = r * G + c * PubKey
		var P, cK ristretto.Point
		P.ScalarMultBase(&r)
		cK.ScalarMult(&pubKeys.keys[i], &prevChallenge)
		P.Add(&P, &cK)
		_, err = buf.Write(P.Bytes())
		if err != nil {
			return ristretto.Scalar{}, err
		}

	}

	for i := 0; i < len(keyImages); i++ {
		r := respsonses[i]

		// P = r * H(K) + c * Ki
		var P, cK ristretto.Point
		var hK ristretto.Point
		hK.Derive(pubKeys.keys[i].Bytes())
		P.ScalarMult(&hK, &r)
		cK.ScalarMult(&keyImages[i], &prevChallenge)
		P.Add(&P, &cK)
		_, err = buf.Write(P.Bytes())
		if err != nil {
			return ristretto.Scalar{}, err
		}
	}

	var challenge ristretto.Scalar
	challenge.Derive(buf.Bytes())

	return challenge, nil
}

func (proof *Proof) calculateKeyImages(skipLastKeyImage bool) []ristretto.Point {
	var keyImages []ristretto.Point

	privKeys := proof.privKeys
	pubKeys := proof.signerPubKeys

	for i := 0; i < len(privKeys); i++ {
		keyImages = append(keyImages, CalculateKeyImage(privKeys[i], pubKeys.keys[i]))
	}

	if !skipLastKeyImage {
		return keyImages
	}

	// Here we assume that there will be atleast one privkey
	// which means there will be atleast one key image
	keyImages = keyImages[:len(keyImages)-1]
	return keyImages
}

func CalculateKeyImage(privKey ristretto.Scalar, pubKey ristretto.Point) ristretto.Point {
	var keyImage ristretto.Point
	keyImage.Set(&pubKey)
	// P = H(xG)
	keyImage.Derive(keyImage.Bytes())
	// P = xH(P)
	keyImage.ScalarMult(&keyImage, &privKey)
	return keyImage
}

func isNumInList(x int, numList []int) bool {
	for _, b := range numList {
		if b == x {
			return true
		}
	}
	return false
}

func readerToPoint(r io.Reader, p *ristretto.Point) error {
	var x [32]byte
	err := binary.Read(r, binary.BigEndian, &x)
	if err != nil {
		return err
	}
	ok := p.SetBytes(&x)
	if !ok {
		return errors.New("point not encodable")
	}
	return nil
}
func readerToScalar(r io.Reader, s *ristretto.Scalar) error {
	var x [32]byte
	err := binary.Read(r, binary.BigEndian, &x)
	if err != nil {
		return err
	}
	s.SetBytes(&x)
	return nil
}
