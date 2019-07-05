package mlsag

import (
	"encoding/binary"
	"errors"
	"io"

	ristretto "github.com/bwesterb/go-ristretto"
)

type Responses []ristretto.Scalar

func (res Responses) Equals(other Responses) bool {

	if len(res) != len(other) {
		return false
	}

	for i := range res {
		ok := res[i].Equals(&other[i])
		if !ok {
			return ok
		}
	}
	return true
}
func (res *Responses) AddResponse(r ristretto.Scalar) {
	*res = append(*res, r)
}

func (res Responses) Len() int {
	return len(res)
}

func (res Responses) Encode(w io.Writer) error {
	for i := range res {
		response := res[i]
		err := binary.Write(w, binary.BigEndian, response.Bytes())
		if err != nil {
			return err
		}
	}
	return nil
}

func (res *Responses) Decode(r io.Reader, numResponses uint32) error {
	if res == nil {
		return errors.New("struct is nil")
	}

	var xBytes [32]byte
	var x ristretto.Scalar
	for i := uint32(0); i < numResponses; i++ {
		err := binary.Read(r, binary.BigEndian, &xBytes)
		if err != nil {
			return err
		}
		x.SetBytes(&xBytes)
		res.AddResponse(x)
	}
	return nil
}
