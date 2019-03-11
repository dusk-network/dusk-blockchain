package transactions

import (
	"errors"
	"io"
)

// TypeInfo is an interface for type-specific transaction information.
type TypeInfo interface {
	Type() TxType
}

func encodeTypeInfo(typeInfo TypeInfo, txType TxType) ([]byte, error) {
	switch txType {
	case CoinbaseType:
		coinbase := typeInfo.(*Coinbase)
		return encodeCoinbaseTransaction(coinbase)
	case BidType:
		bid := typeInfo.(*Bid)
		return encodeBidTransaction(bid)
	case StakeType:
		stake := typeInfo.(*Stake)
		return encodeStakeTransaction(stake)
	case StandardType:
		standard := typeInfo.(*Standard)
		return encodeStandardTransaction(standard)
	case TimelockType:
		timelock := typeInfo.(*Timelock)
		return encodeTimelockTransaction(timelock)
	case ContractType:
		contract := typeInfo.(*Contract)
		return encodeContractTransaction(contract)
	default:
		return nil, errors.New("unknown transaction type")
	}
}

func decodeTypeInfo(r io.Reader, txType TxType) (TypeInfo, error) {
	switch txType {
	case CoinbaseType:
		return decodeCoinbaseTransaction(r)
	case BidType:
		return decodeBidTransaction(r)
	case StakeType:
		return decodeStakeTransaction(r)
	case StandardType:
		return decodeStandardTransaction(r)
	case TimelockType:
		return decodeTimelockTransaction(r)
	case ContractType:
		return decodeContractTransaction(r)
	default:
		return nil, errors.New("unknown transaction type")
	}
}
