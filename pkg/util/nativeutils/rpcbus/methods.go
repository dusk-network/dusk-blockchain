package rpcbus

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
)

func (rb *RPCBus) LoadWallet(password string) (string, error) {

	buf := new(bytes.Buffer)
	if err := encoding.WriteString(buf, password); err != nil {
		return "", err
	}

	req := NewRequest(*buf, -1)
	pubKeyBuf, err := rb.Call(LoadWallet, req)
	if err != nil {
		return "", err
	}

	var pubKey string
	pubKey, err = encoding.ReadString(&pubKeyBuf)
	if err != nil {
		return "", err
	}

	return pubKey, nil
}

func (rb *RPCBus) CreateFromSeed(seed, password string) (string, error) {

	buf := new(bytes.Buffer)
	if err := encoding.WriteString(buf, seed); err != nil {
		return "", err
	}

	if err := encoding.WriteString(buf, password); err != nil {
		return "", err
	}

	req := NewRequest(*buf, -1)
	pubKeyBuf, err := rb.Call(CreateFromSeed, req)
	if err != nil {
		return "", err
	}

	var pubKey string
	pubKey, err = encoding.ReadString(&pubKeyBuf)
	if err != nil {
		return "", err
	}

	return pubKey, nil
}

func (rb *RPCBus) CreateWallet(password string) (string, error) {

	buf := new(bytes.Buffer)
	if err := encoding.WriteString(buf, password); err != nil {
		return "", err
	}

	req := NewRequest(*buf, -1)
	pubKeyBuf, err := rb.Call(CreateWallet, req)
	if err != nil {
		return "", err
	}

	var pubKey string
	pubKey, err = encoding.ReadString(&pubKeyBuf)
	if err != nil {
		return "", err
	}

	return pubKey, nil
}

func (rb *RPCBus) SendBidTx(amount, lockTime uint64) ([]byte, error) {

	buf := new(bytes.Buffer)
	if err := encoding.WriteUint64LE(buf, amount); err != nil {
		return nil, err
	}

	if err := encoding.WriteUint64LE(buf, lockTime); err != nil {
		return nil, err
	}

	req := NewRequest(*buf, -1)
	txIdBuf, err := rb.Call(SendBidTx, req)
	if err != nil {
		return nil, err
	}

	return txIdBuf.Bytes(), nil
}

func (rb *RPCBus) SendStakeTx(amount, lockTime uint64) ([]byte, error) {

	buf := new(bytes.Buffer)
	if err := encoding.WriteUint64LE(buf, amount); err != nil {
		return nil, err
	}

	if err := encoding.WriteUint64LE(buf, lockTime); err != nil {
		return nil, err
	}

	req := NewRequest(*buf, -1)
	txIdBuf, err := rb.Call(SendStakeTx, req)
	if err != nil {
		return nil, err
	}

	return txIdBuf.Bytes(), nil
}

func (rb *RPCBus) SendStandardTx(amount uint64, pubkey string) ([]byte, error) {

	buf := new(bytes.Buffer)

	if err := encoding.WriteUint64LE(buf, amount); err != nil {
		return nil, err
	}

	if err := encoding.WriteString(buf, pubkey); err != nil {
		return nil, err
	}

	req := NewRequest(*buf, -1)
	txIdBuf, err := rb.Call(SendStandardTx, req)
	if err != nil {
		return nil, err
	}

	return txIdBuf.Bytes(), nil
}

func (rb *RPCBus) GetBalance() (uint64, error) {
	buf := new(bytes.Buffer)
	req := NewRequest(*buf, -1)
	balanceBuf, err := rb.Call(GetBalance, req)
	if err != nil {
		return 0, err
	}

	var balance uint64
	if err := encoding.ReadUint64LE(&balanceBuf, &balance); err != nil {
		return 0, err
	}

	return balance, nil
}
