package transactions

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/toghrulmaharramov/dusk-go/crypto"
)

func TestNewStealth(t *testing.T) {
	byt32 := []byte{1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4}

	// Input
	sig, _ := crypto.RandEntropy(2000)
	in := Input{
		byt32,
		byt32,
		1,
		sig,
	}

	// Output
	out := Output{
		200,
		byt32,
	}

	// Type attribute
	ta := TypeAttributes{
		[]Input{in},
		byt32,
		[]Output{out},
	}

	R, _ := crypto.RandEntropy(32)
	s := Stealth{
		1,
		1,
		R,
		ta,
	}

	buf := new(bytes.Buffer)

	err := s.Encode(buf)
	if err != nil {
		t.Fail()
	}

	fmt.Println(buf.Bytes())
}
