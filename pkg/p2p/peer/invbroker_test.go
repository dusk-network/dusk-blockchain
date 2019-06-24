package peer_test

import (
	"bufio"
	"bytes"

	"net"
	"testing"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database/lite"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/tests/helper"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/chainsync"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/dupemap"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/peermsg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/processing"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

func TestSendInv(t *testing.T) {

	// Send topics.Inv
	b := helper.RandomBlock(t, 1, 1)
	encoded, _ := sendWithInvPayload(t, b.Header.Hash, topics.Inv)

	// Expect topics.GetData to be received in return
	buf := assertMsg(t, encoded.Bytes(), topics.GetData)

	// Assert the output
	invPayload := &peermsg.Inv{}
	err := invPayload.Decode(bytes.NewBuffer(buf))
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(invPayload.InvList[0].Hash, b.Header.Hash) {
		t.Error("hash from topics.GetData is not equal to the hash from topics.Inv")
	}

	if len(invPayload.InvList) > 1 {
		t.Error("expecting only one object")
	}
}

func TestSendGetData(t *testing.T) {
	drvr, err := database.From(lite.DriverName)
	if err != nil {
		t.Fatal(err)
	}

	db, err := drvr.Open("", protocol.TestNet, false)
	if err != nil {
		t.Fatal(err)
	}

	defer db.Close()

	// Send topics.GetData
	// Expect topics.Block to be received in return
	hashes, blocks := generateBlocks(t, 1, db)
	encoded, _ := sendWithInvPayload(t, hashes[0], topics.GetData)

	buf := assertMsg(t, encoded.Bytes(), topics.Block)

	// assert the output
	b := &block.Block{}
	err = b.Decode(bytes.NewBuffer(buf))
	if err != nil {
		t.Fatal(err)
	}

	if !blocks[0].Equals(b) {
		t.Error("block from topics.Block is not equal to the block from topics.GetData")
	}
}

func assertMsg(t *testing.T, encodedMsg []byte, expected topics.Topic) []byte {

	eb := wire.NewEventBus()
	cs := chainsync.LaunchChainSynchronizer(eb, protocol.TestNet)

	client, serv := net.Pipe()

	go func() {
		dupeMap := dupemap.NewDupeMap(5)
		peerReader, err := peer.NewReader(serv, protocol.TestNet, dupeMap, eb, cs)
		if err != nil {
			t.Fatal(err)
		}
		peerReader.ReadLoop()
	}()

	r := bufio.NewReader(client)

	if _, err := client.Write(encodedMsg); err != nil {
		t.Fatal(err)
	}

	bs, err := r.ReadBytes(0x00)
	if err != nil {
		t.Fatal(err)
	}

	decoded := processing.Decode(bs)

	// Remove magic bytes
	if _, err := decoded.Read(make([]byte, 4)); err != nil {
		t.Fatal(err)
	}

	// Check for correctness of topic
	var topicBytes [15]byte
	if _, err := decoded.Read(topicBytes[:]); err != nil {
		t.Fatal(err)
	}

	topic := topics.ByteArrayToTopic(topicBytes)
	if topic != expected {
		t.Fatalf("unexpected topic %s, expected GetData", topic)
	}

	return decoded.Bytes()

}

func sendWithInvPayload(t *testing.T, hash []byte, topic topics.Topic) (*bytes.Buffer, *peermsg.Inv) {

	msg := &peermsg.Inv{}
	msg.InvList = make([]peermsg.InvVect, 1)
	msg.InvList[0].Type = peermsg.InvTypeBlock
	msg.InvList[0].Hash = hash

	buf := new(bytes.Buffer)
	if err := msg.Encode(buf); err != nil {
		t.Fatal(err.Error())
	}

	withTopic, err := wire.AddTopic(buf, topic)
	if err != nil {
		t.Fatal(err.Error())
	}

	g := processing.NewGossip(protocol.TestNet)
	encoded, err := g.Process(withTopic)
	if err != nil {
		t.Fatal(err.Error())
	}

	return encoded, msg
}
