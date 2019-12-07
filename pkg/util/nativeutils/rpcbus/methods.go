package rpcbus

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
)

type method uint8

const (
	GetLastBlock method = iota
	GetMempoolTxs
	GetMempoolTxsBySize
	SendMempoolTx
	VerifyCandidateBlock
	CreateWallet
	CreateFromSeed
	LoadWallet
	SendBidTx
	SendStakeTx
	SendStandardTx
	GetBalance
	GetAddress
)

func MarshalConsensusTxRequest(buf *bytes.Buffer, amount, lockTime uint64) error {
	if err := encoding.WriteUint64LE(buf, amount); err != nil {
		return err
	}

	return encoding.WriteUint64LE(buf, lockTime)
}
