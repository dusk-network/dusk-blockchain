// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package message_test

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/checksum"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/stretchr/testify/assert"
)

//nolint
func TestWireTransaction(t *testing.T) {
	assert := assert.New(t)

	hexWire := "a7080000000000000000000001000000000000000000000000000000b9e6f5c30a0100000001000000520800000100000000000000cc8f1b3e111f4203cb1f38b7d43b14713e59678264179b7c29797ba12daa35570200000000000000010c8088b9e8c9d06915673d4d94fc76348fb7ce7503e8587f30caea67ab8379b815ce6aba274054f337bdd92d9411d8be3f282b05e3c6d42e8eea9f3215b8de33b96a3c7c1dbcb4d8cdd8ef13e50e84cf6480116311677676269d3e662cea608c5a3479e042102a78621252a37f2d99e6824e17a2b11597147d1adf4624e7d436ffffffffffffffff997ebe7877346dc48137c1d115176c60c5dbf0ea77dd8cdca0cfbc0f3d90304ecb5b2b3d60a2b9d4df4a999ef3a768f8bd75c75aac343bff35bed7cfb2e3513315e8ece73c24ca0c97bda403149dcf9fea1c8827b682c1bbe089c8d10355c45e01e549d068cb470cbefe6fddd3b2d8aacfa5a76805e725d5394e882a79d157695ec48dcb7e531ccc3b334ae122d4fd40e242e7d8a85fdb82bd4c9e9621a9a60d042dbbaec8a2acb879b48d311f1264b1aafe6bf26ccc0bb250af7a2e19e8dcdc3851f382c509fb449a701a93c9489ae97bae88feaebe38fc6c128dc4b286724c10ffffffffffffffff14b611da24f94e89dd03121410f05b53c52cfc785da3c8d59bb65d07b78ab0241c6a8f3ffadc251790b9f78a31b82246c883cbfa1330337bd094045c01dcca2a7de1eadf6f1f7116169ed9dd10541a407035bb8fe834a973d8176f51f07a8435fee6a01aa94b675366ed1b054b8091542329dd1538bcec8a7503906281f0b61200ca9a3b000000000200000000000000d85dbd596fc0476c779f3e2e7b5e58b732cb71f9ca056a8828cf845885a22f17848a28b1224942eb4222b7c43fc01e60529c7ee5fab115f3802c91179d0edfa19851d4394c5da06a86f955b2bd1305672e61a9569b5e024f03c957d4160d3d23fad4651d0897d60d89845c58baee90dbb291366e711628910673b9f3eedaaec355d87e2b2619a6809157bf01d3579145794a2b10e5e0f23d053e48a699ad318d80d2e737ca67e32f0848724907f3a847befe125d83031fc249cc24d489bee3cca6dfba0129d5578102c594b72631a13797cc0413391a5a1886c7536e6fdc0c489dfdbc00baba13e05157a7ab7273523dbb98d34c06e3a058424f361aad4a8fbda04b3327dbf973a2fc07d54445ebe6651b2e35a3f5c983dad6f05599505d20e8049ab8b6a8f099304dbc4badb806e2e8b02f90619eacef17710c48c316cddd0889badea8613806d13450208797859e6271335cda185bbfc5844358e701c0ca03ad84e86019661d4b29336d10be7f2d1510cb65478f0ea3e0baea5d49ff962bcccdcf4396a0b3cfed0f1b8c5537b148f88f31e782f30be64807cad8900706b18a31cce9a743694b0abf94d6ff32789e870b3b70970bc2a01b69faea5a6dfc3514b4d6cf831dd715429cb3c9c3c9011422260233eab35f30dec5415fe06f9a22e5e4847cde93f61e896ebeec082ced1e65b7bf5dfe6f6dd064d2649580ae5ec6b09934167cdd0efc24150dee406c18dc4d6def110c74049a3f14c7d2b019606518ab91cba648915908d032c33cd3a6c07bfb908902c5a8bd55ed5fb25582659a9f4fb82aedba03c6946823b020ff8fad039772696c1b58a3434a5c53f5b6670943e90ccf49fb24d88929f467341cd68978082969dfc75ccdf161e1340bb3d66633b52703b2efd6cf769395fa892f5738cf5dee96afe27fe085bed54dd607bc0f0b3fe5fd5e83f1a18ed9e3457ac28bc6a49224c20f17d63fbc38f2d3e49af4f108407a9523e55fc1e89a2c221b0d15a993a3856a9f9618655555f7828734da3193ad2353c81a6f0720e90dbc62a8dcdd1e117b8f6addd574a6c483a5bebb06255e9614ff22ce4ac848de8ee8df47bd133fbd5f46bf9bf9a56e80d6e411cf2803186dad1a7cd9176ba85dff17e29471fb1c6f3a9304630e190406857e511c93711eca6a472f89005ddef430f0df953dcf5a3751bddaf39da32e25a87b1f41cc23f14b25ea9e0289785520696b0a82d6a23a19eb11ca32021c414ba83f0d4012933a4a962826e7185f21f440c8b08c1adf58aec9daee1c8e15e607239e819fc5dea80c697e800a1a18acd235789fb9dfee43f3e8a51ba190656ca8ee9dc7ed1cbfce26a0deb7563f52292f3f6bef6360095b1fa416afa01640ddbabbd3b8fc15223d50c0cdc80cb846947b80408764fab356051d2783e2a9e54917cfaab223c75dd8d5187841fbe93fc79bbc1d63ffffce68ae16c3b4ef3bd92d87bec21f2f958ab4f91535f10c50ef186e3a4d2a43b8060ac15b9ef21256e52123862563540c14d9d0904c20c70d2c5915e352b582f7ee0dfe3338658c1e7245b651428799705d9b76847e9fc8a872ef3aae9c978ca64e3f5f11dd7d49decaad5c299680e7478ddc9651d8578774431b46cc701601af616f9c7323ce76fcd1c6055f7d02652c9a2354ad21ebfd1df37d5254609e3d38666940a2a6dd21c59400bf444f8b297203243de4099b1c8640fb43849f160cdab42a52e0a107df5db400819f7587957f07d72cb498ae97aa6d1e67ae2900ff56f7378f742e04fcdedd2a72ef20aea340f9f65cff2bedc1362733170906a443a1964bdc59c245808014604e2fc9c9f23ecc590da6bedcb81c69ef8f369d69a0c9c663e0faccefde8bf848224166c59b49eb9a58f8fb38bdb42f6b33b5470378bfe21a980b1d78a8da4c32b4f380127bdd6a9c0c96f1b3ee4c0bbc69fa312e7a77560ad2eafdc97017ff9e51da30ee8e2acfaef091236c4c6cf66e2f43129d70744812d2eafdc97017ff9e51da30ee8e2acfaef091236c4c6cf66e2f43129d707448126981ddc905c11356d461b7ccc828dc1ac8e3c92cc9ba3619ee76f9150095a75304d64fd0d2d436f18e6881aae6b7d99bed17078b8f508f0cf4bb2dbd3e7f7871170c739f9d9ea4404bff4066c3ed34d6a52245965b485b766344a380f65e5d2800000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
	hexTxHash := "63155db32cdb892e74465950c5fc0ab0e25c218855e141430333cc68cb78e208"

	txHash, err := hex.DecodeString(hexTxHash)
	if err != nil {
		t.Fatalf("Unable to decode hash hex: %v", err)
	}

	wire, err := hex.DecodeString(hexWire)
	if err != nil {
		t.Fatalf("Unable to decode tx hex: %v", err)
	}

	// buffer := bytes.NewBuffer(wire)

	gossip := protocol.NewGossip(protocol.DevNet)
	// frame := gossip.(buffer)

	reader := bytes.NewReader(wire)

	// read message (extract length and magic)
	b, err := gossip.ReadMessage(reader)
	if err != nil {
		t.Fatalf("Cannot extract length and magic: %v", err)
	}
	// extract checksum
	m, cs, err := checksum.Extract(b)
	if err != nil {

		t.Fatalf("error extracting message and cs: %v", err)
		return
	}
	// verify checksum
	if !checksum.Verify(m, cs) {
		t.Fatalf("invalid checksum: %v", err)
		return
	}

	buffer := bytes.NewBuffer(m)

	message, err := message.Unmarshal(buffer, config.KadcastInitHeader)
	if err != nil {
		t.Fatalf("Unable to unmarshal: %v", err)
	}
	tx := message.Payload().(*transactions.Transaction)

	assert.EqualValues(1, tx.Version, "tx Version mismatch")
	assert.EqualValues(transactions.Transfer, tx.TxType, "txType is not TRANSFER")

	decoded, err := tx.Decode()
	if err != nil {
		t.Fatalf("Unable to unmarshal: %v", err)
	}

	assert.EqualValues(1000000000, decoded.Fee.GasLimit, "unexpected GasLimit")
	assert.EqualValues(2, decoded.Fee.GasPrice, "unexpected GasPrice")

	hash, err := decoded.Hash(tx.TxType)
	if err != nil {
		t.Fatalf("Unable to hash transaction: %v", err)
	}

	assert.EqualValues(txHash, hash, "hash mismatch")
}

//nolint
func TestCoinbaseTransaction(t *testing.T) {
	assert := assert.New(t)

	hexPayload := "007d2f25b968f9d81cb8d53cc4149888c8f9dc28b8746380c9f54c9dbec55548a0dc95b7941a61534f5b12733a8ede7869d21ee3108d95e7c3ad2cf95a5e3502248f718574d92c255f0e4a3ac0394baf17d45d87d621287edd07674b8809da13cb1ee914c33fb33d2b6b39ad18ffc7d816102c23421e7b99a68b71a08a43826bd20e00000000000000c8fbac010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002f35c3ada6983ea2e312d68fee8e8e50d590cdc846164931a6edeaf0c3fe07493a866915a98c56251bc5bb561e97f7bdf687003c4a742fa42aa6697c2e5afa10eedba04eec9b41e4b800b80be19b2a506c99360f41b8663ae5e7bd4d93f8f79e09455d26c1f02f3c71364f1e471110dfa2ad406bc0fc437c827fbd48193b67c00f000000000000000eda140f0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
	hexTxHash := "bdcadedb29422e70d2f39d5901cd13f7ddeade980198aa2a79be2a16e080890a"

	txHash, err := hex.DecodeString(hexTxHash)
	if err != nil {
		t.Fatalf("Unable to decode hash hex: %v", err)
	}

	payload, err := hex.DecodeString(hexPayload)
	if err != nil {
		t.Fatalf("Unable to decode tx hex: %v", err)
	}

	txdummy := transactions.NewTransaction()
	txdummy.TxType = 0
	txdummy.Payload.Data = payload

	decoded, err := txdummy.Decode()
	if err != nil {
		t.Fatalf("Unable to decode coinbase: %v", err)
	}
	assert.EqualValues(decoded.Anchor, make([]byte, 32), "Anchor should be empty")
	assert.Nil(decoded.Call, "Call should not be present")
	assert.Nil(decoded.Crossover, "Crossover should not be present")
	assert.Zero(decoded.Fee.GasLimit, "GasLimit should be 0")
	assert.Zero(decoded.Fee.GasPrice, "GasPrice should be 0")
	assert.Empty(decoded.Nullifiers, "Nullifiers should not be present")
	assert.EqualValues(uint64(28113864), decoded.Notes[0].DecodeTxAmount())
	assert.EqualValues(uint64(253024782), decoded.Notes[1].DecodeTxAmount())

	hash, err := decoded.Hash(txdummy.TxType)
	if err != nil {
		t.Fatalf("Unable to calculate hash for coinbase: %v", err)
	}

	assert.EqualValues(txHash, hash, "hash mismatch")
}
