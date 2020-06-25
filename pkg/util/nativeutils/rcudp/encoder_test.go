package rcudp

import (
	"bytes"
	"testing"

	crypto "github.com/dusk-network/dusk-crypto/hash"
)

func TestEncodeRFC5053(t *testing.T) {

	minLength := BlockSize*4 + 1
	messagesNum := 10001

	data, err := crypto.RandEntropy(uint32(messagesNum))
	if err != nil {
		t.Fatal(err)
	}

	for i := minLength; i < len(data); i++ {

		// Dummy message
		message := make([]byte, i)
		copy(message, data[0:i])

		// messageCopy is needed because encoding is destructive to the message array
		messageCopy := make([]byte, i)
		copy(messageCopy, data[0:i])

		// Encode
		w, err := NewEncoder(messageCopy, BlockSize, 1, symbolAlignmentSize)
		if err != nil {
			t.Fatal(err)
		}
		blocks := w.GenerateBlocks()

		// Decode
		d := NewDecoder(w.NumSourceSymbols,
			symbolAlignmentSize,
			w.TransferLength(),
			int(w.PaddingSize))

		var decoded []byte
		for i := 0; i < len(blocks); i++ {
			decoded = d.AddBlock(blocks[i])
			if decoded != nil {
				// Decoded after N blocks
				//t.Logf("DECODED after %d blocks --------", i+1)
				break
			}
		}

		// Compare
		if decoded != nil {
			if !bytes.Equal(message, decoded) {
				t.Fatalf("Decoding result must equal %d vs %d", len(message), len(decoded))
			}
			// Message completely decoded
			// t.Logf("Decoded original message of size %d", len(message))
		} else {
			t.Fatal("Decoding determined failed")
		}
	}
}
