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
	GetMempoolView
	SendMempoolTx
	VerifyCandidateBlock
	CreateWallet
	CreateFromSeed
	LoadWallet
	SendBidTx
	SendStakeTx
	SendStandardTx
	GetBalance
	GetUnconfirmedBalance
	GetAddress
	GetTxHistory
	GetLastCertificate
	GetCandidate
	GetRoundResults
	AutomateConsensusTxs
	GetSyncProgress
	IsWalletLoaded
	StartProfile
	StopProfile
	RebuildChain
	ClearWalletDatabase
)

func MarshalConsensusTxRequest(buf *bytes.Buffer, amount, lockTime uint64) error {
	if err := encoding.WriteUint64LE(buf, amount); err != nil {
		return err
	}

	return encoding.WriteUint64LE(buf, lockTime)
}
