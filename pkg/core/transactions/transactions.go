package transactions

import (
	"bytes"
	"fmt"
	"io"
)

// FromReader will decode into a slice of transactions
func FromReader(r io.Reader, numOfTXs uint64) ([]Transaction, error) {

	var txs []Transaction

	// Load all bytes into a ReadSeeker
	buf := new(bytes.Buffer)
	buf.ReadFrom(r)
	reader := bytes.NewReader(buf.Bytes())

	for i := uint64(0); i < numOfTXs; i++ {

		// Peak the type from the first byte
		txtype, err := reader.ReadByte()
		if err != nil {
			return nil, err
		}
		err = reader.UnreadByte()
		if err != nil {
			return nil, err
		}

		switch TxType(txtype) {
		case StandardType:
			tx := &Standard{}
			err := tx.Decode(reader)
			if err != nil {
				return nil, err
			}
			txs = append(txs, tx)
		case TimelockType:
			tx := &TimeLock{}
			err := tx.Decode(reader)
			if err != nil {
				return nil, err
			}
			txs = append(txs, tx)
		case BidType:
			tx := &Bid{}
			err := tx.Decode(reader)
			if err != nil {
				return nil, err
			}
			txs = append(txs, tx)
		case StakeType:
			tx := &Stake{}
			err := tx.Decode(reader)
			if err != nil {
				return nil, err
			}
			txs = append(txs, tx)
		case CoinbaseType:
			tx := &Coinbase{}
			err := tx.Decode(reader)
			if err != nil {
				return nil, err
			}
			txs = append(txs, tx)
		default:
			return nil, fmt.Errorf("unknown transaction type: %d", txtype)
		}

	}

	return txs, nil
}
