package helper

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
)

// TxsToReader converts a slice of transactions to an io.Reader
func TxsToReader(t *testing.T, txs []transactions.Transaction) io.Reader {
	buf := new(bytes.Buffer)

	for _, tx := range txs {
		err := tx.Encode(buf)
		if err != nil {
			assert.Nil(t, err)
		}
	}

	return bytes.NewReader(buf.Bytes())
}

// RandomSlice returns a random slice of size `size`
func RandomSlice(t *testing.T, size uint32) []byte {
	randSlice := make([]byte, size)
	_, err := rand.Read(randSlice)
	assert.Nil(t, err)
	return randSlice
}

type SimpleStreamer struct {
	*bufio.Reader
	*bufio.Writer
}

func NewSimpleStreamer() *SimpleStreamer {
	r, w := io.Pipe()
	return &SimpleStreamer{
		Reader: bufio.NewReader(r),
		Writer: bufio.NewWriter(w),
	}
}

func (ms *SimpleStreamer) Write(p []byte) (n int, err error) {
	n, err = ms.Writer.Write(p)
	if err != nil {
		return n, err
	}

	return n, ms.Writer.Flush()
}

func (ms *SimpleStreamer) Read() ([]byte, error) {
	// check the event
	// discard the topic first
	topicBuffer := make([]byte, 15)
	if _, err := ms.Reader.Read(topicBuffer); err != nil {
		return nil, err
	}

	// now unmarshal the event
	buf := make([]byte, ms.Reader.Buffered())
	if _, err := ms.Reader.Read(buf); err != nil {
		return nil, err
	}

	return buf, nil
}

func (ms *SimpleStreamer) Close() error {
	return nil
}
