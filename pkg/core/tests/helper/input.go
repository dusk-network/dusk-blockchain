package helper

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-crypto/mlsag"
	"github.com/dusk-network/dusk-wallet/v2/transactions"
	"github.com/stretchr/testify/assert"
)

const respAmount uint32 = 7

// RandomInput returns a random input for testing
func RandomInput(t *testing.T) *transactions.Input {
	amount := ristretto.Scalar{}
	amount.Rand()
	privKey := ristretto.Scalar{}
	privKey.Rand()
	mask := ristretto.Scalar{}
	mask.Rand()

	sigBuf := randomSignatureBuffer(t)

	in := transactions.NewInput(amount, privKey, mask)
	in.Signature = &mlsag.Signature{}
	if err := in.Signature.Decode(sigBuf, true); err != nil {
		t.Fatal(err)
	}

	in.KeyImage.Rand()
	return in
}

// RandomInputs returns a slice of inputs of size `size` for testing
func RandomInputs(t *testing.T, size int) transactions.Inputs {
	var ins transactions.Inputs

	for i := 0; i < size; i++ {
		in := RandomInput(t)
		assert.NotNil(t, in)
		ins = append(ins, in)
	}

	return ins
}

func randomSignatureBuffer(t *testing.T) *bytes.Buffer {
	buf := new(bytes.Buffer)
	// c
	c := ristretto.Scalar{}
	c.Rand()
	if _, err := buf.Write(c.Bytes()); err != nil {
		t.Fatal(err)
	}

	// r
	rBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(rBytes, 1)
	if _, err := buf.Write(rBytes); err != nil {
		t.Fatal(err)
	}

	// responses
	respBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(respBytes, respAmount)
	if _, err := buf.Write(respBytes); err != nil {
		t.Fatal(err)
	}

	for i := uint32(0); i < respAmount; i++ {
		writeRandomScalar(t, buf)
	}

	// pubkeys
	for i := uint32(0); i < respAmount; i++ {
		writeRandomPoint(t, buf)
	}

	return buf
}

func fixedSignatureBuffer(t *testing.T) *bytes.Buffer {
	buf := new(bytes.Buffer)
	c := ristretto.Scalar{}
	c.SetOne()
	if _, err := buf.Write(c.Bytes()); err != nil {
		t.Fatal(err)
	}

	// r
	rBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(rBytes, 1)
	if _, err := buf.Write(rBytes); err != nil {
		t.Fatal(err)
	}

	respAmount := uint32(1)

	// responses
	respBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(respBytes, respAmount)
	if _, err := buf.Write(respBytes); err != nil {
		t.Fatal(err)
	}

	for i := uint32(0); i < respAmount; i++ {
		s := ristretto.Scalar{}
		s.SetOne()
		if _, err := buf.Write(s.Bytes()); err != nil {
			t.Fatal(err)
		}
	}

	// pubkeys
	for i := uint32(0); i < respAmount; i++ {
		p := ristretto.Point{}
		p.SetZero()
		if _, err := buf.Write(p.Bytes()); err != nil {
			t.Fatal(err)
		}
	}

	return buf
}
