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

	hexWire := "9f08000000000000406e737400000000000000005a65065d0a0100000001000000520800000100000000000000de45f7c56d19993299e46e28cdefe80444bfbcc558679173d8a5c068f6abd3060200000000000000014dabcafdc6d211035a03d688cc83e09e01a89b58d5604aae7fa70196d58da6c315ce6aba274054f337bdd92d9411d8be3f282b05e3c6d42e8eea9f3215b8de33b96a3c7c1dbcb4d8cdd8ef13e50e84cf6480116311677676269d3e662cea608c5a3479e042102a78621252a37f2d99e6824e17a2b11597147d1adf4624e7d436ffffffffffffffff9974f2345e356dc48137c1d115176c60c5dbf0ea77dd8cdca0cfbc0f3d90304ecb5b2b3d60a2b9d4df4a999ef3a768f8bd75c75aac343bff35bed7cfb2e35133befb1d16048306e60895b1b5fc7d270b33542897cec4bca5d0b9ae826949960901e549d068cb470cbefe6fddd3b2d8aacfa5a76805e725d5394e882a79d157695ec48dcb7e531ccc3b334ae122d4fd40e242e7d8a85fdb82bd4c9e9621a9a60d042dbbaec8a2acb879b48d311f1264b1aafe6bf26ccc0bb250af7a2e19e8dcdc3851f382c509fb449a701a93c9489ae97bae88feaebe38fc6c128dc4b286724c10ffffffffffffffff14b611da24f94e89dd03121410f05b53c52cfc785da3c8d59bb65d07b78ab0241c6a8f3ffadc251790b9f78a31b82246c883cbfa1330337bd094045c01dcca2a7de1eadf6f1f7116169ed9dd10541a407035bb8fe834a973d8176f51f07a8435bb463366c465e470f1b7501917b906a9db5607cb3ef57c1b8bd7c9725063647300ca9a3b000000000100000000000000d85dbd596fc0476c779f3e2e7b5e58b732cb71f9ca056a8828cf845885a22f17848a28b1224942eb4222b7c43fc01e60529c7ee5fab115f3802c91179d0edfa1a0e6aa001633252dc50d9a3e9e3dc7b23365251db481b45ec8f1cf4a4a7e3f036c3ee0b4dc438d98157fb886cce964ba8c1265b99e8fa21418e6ab8f56520f0a704db35fa84a53eadf24e8a4f8a180aa2516e69a47ec10cd2dabde3d5658a3fe97f327062fab1f3b7ceb1c7150a12c02d78ce32d51b885514603ebfdee8cac69700b5ab2c1bf08b690111e0c953a088c91782f75fc2a8b1c4c5bf4aacff3ecafe94311cf5e2c94d1ac441f7c01b2ddcde8c0facb7f746d91addab1282d06506da02d1446630ee85e39d9abecf2aa3c9ba53289bd03332a8182bee045a5dd8115b26fcc34bc637f2d21797f4dae564d79b199c25c51ea17e757ba45ea39a402afd63862e6e353e459a00cd068ac2b759cd6352bf540dc74118bb1fe821c4922f690cc8698b7f162c3932524f8f17941afd213fa9899cf3d58922534a9c21b0667f74307e9b3abe4aedce41f1af6621daeb9648a588c5a1ec0df7a423932b191fee9226e57f59f83256cb15c952ee2bdf472219a873b12f638b6d51ca06e2cbf9783e3feec7f213ed06f7c45c0240858204f8ba25e36f5877be3ade9d68b1b6e1cf49651e11f44e8a11984b29a5f149bdd91f3624e2107579274c406fbda788f5cabcb472a5fe93243b76bcbfc946cc8da600cd7458e428736fe4545259b2a2eda8aab3c187836afae79ec333c9918431b7f6ac024d96d9d83dbbda9084c95ca4999a129f9a6bdc62b09bda93a850e0807b5ea9844531571b1fd99cafffb20614fa7c48ecfcde95184769bf34bcc3fdc4273be119bb5c29ee1290e1e920634543fb7e4f72f439a12ca8888cc6febd7fb1158abb02023fa85ec5c0670be51e5a87fde49e0dd5dfa9e7d2c8679e6ced9b212852c88cce5d8333fc9ce4c91f2d00f80a0b1412ec7f6340a9f237b47e25eb42074c528c2e733ad1fcc7186ffff4a352c848fee121fa3262ea4fd53ecc9e79e2390b6519be98404248cb7ab11bd9fc004250092d7cccff76ab21f7371e7716a844728d768a2872caa191fad3932dd69bba6ebbb2666e75150f967d757d6c8b316ab1275bb9344682f851bef8e205cbdf42f075773275198d982cad99d0a4aff45f612e27ddbaf46744ce527cf62ef7f72bca5e7990997697611787ef1827a071447059954fa85f2eb23906a3e6aa78e514e104a80e63960ab77a206edb830be39c37c9e5d727cf22ab6588b1365ca05f7c21d51ceb1b56720f3c4f4302cff1044facadb7c59c13907ba9bfda92dce75a67b608278dfd6b3ec90160cc07cb3801e70dcee28684e10076d32d82ecd26ebcb445fd5998baf983e5ac13d70263b881aacc01a39223cda26281a3afd110164a9dad8319b1103211573cc87d9daa94120c1083aa325828d99bcd079b771d35b1cdbb2c8a52be5c2dabbb7363c0e5b4b2e0e59699f6221f8565b1faf700b98f63c11581a7ce0036946f760641fbce4e4671cd27b3c6946c15254af122af572604611da246e86ea9d74217d82ebe962b257e472ea27abf873696689bc35bab09ecdecafe9055db6dbe931c5376bd7c42047bd0aaf67407de9c901dc1a57b2d606a042303cb4d0f9dfdf7f13666f9a3cd927a950d5c9564e2e8e4c3ebe4ea5f2df933d0e58fc71496b93cbc736d69388590f16bf3769726801580cd676278237800283972d11e322e87954a42af26917e563b2ca5ab16ec943cd6301d32ea1993630112ff3d02946a0877a2ca44b444f6173358c7f85cd1e65c92dfc07bc4d5c2fd07fea0b319e24cfe455219673a2657c10d1d9cca2606ad0f3d5a42a583a9409bf601ac08b0810d4487d274de0dcbc9f06e7c993d218b5393a01cd439c750c0d42c2e13213538a24144528eb3ea2571036f2d8a131b0ce75022a4d92849247d53e253bc5659272169f814b44b426886b2ff2d8a131b0ce75022a4d92849247d53e253bc5659272169f814b44b426886b2f9daf3d9e051ffbe1630ffc5efb9c0cf030057bb33ce70ad7276a16b86d43784d16ae1850b8b67629fe09e387d9018366cb5a957b5b8201a0e8319b471cc3274bf5ec945e23c62e7f99a3b83b19caffe2900bbf032c49782dd5764e484a83c37000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
	hexTxHash := "7e466ac297e8a493a4918fdab05e16cd633b0aefb50caea4ef614b92ecb94307"

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

	decoded, err := tx.Payload.Decode()
	if err != nil {
		t.Fatalf("Unable to unmarshal: %v", err)
	}

	assert.EqualValues(1000000000, decoded.Fee.GasLimit, "unexpected GasLimit")
	assert.EqualValues(1, decoded.Fee.GasPrice, "unexpected GasLimit")

	hash, err := decoded.Hash()
	if err != nil {
		t.Fatalf("Unable to hash transaction: %v", err)
	}

	assert.EqualValues(txHash, hash, "hash mismatch")
}
