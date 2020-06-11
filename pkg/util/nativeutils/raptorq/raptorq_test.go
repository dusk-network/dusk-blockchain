// +build

package raptorq

import (
	"bytes"
	"net"
	"testing"
	"time"

	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/sirupsen/logrus"
)

const (
	packetLen = 2000 * 1000
)

func TestRaptorQ(t *testing.T) {

	logrus.SetLevel(logrus.TraceLevel)

	raddr, err := net.ResolveUDPAddr("udp4", "127.0.0.1:4444")
	if err != nil {
		t.Error(err)
	}

	// Start remote service
	r, err := NewUDPReader(raddr)
	if err != nil {
		t.Error(err)
	}

	go r.Serve()

	time.Sleep(1 * time.Second)

	laddr, err := net.ResolveUDPAddr("udp4", "127.0.0.1:0")
	if err != nil {
		t.Error(err)
	}

	// Send a single sourceObject
	sourceObject, err := crypto.RandEntropy(packetLen)
	if err != nil {
		t.Fatal(err)
	}

	begin := time.Now().UnixNano()

	go SendUDP(laddr, raddr, sourceObject, 5)

	var decodedObject []byte
	select {
	case id := <-r.ReadySourceObjectChan:
		decodedObject, _, err = r.CollectDecodedSourceObject(id)
		if err != nil {
			t.Error(err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("raptorq timeout error")
	}

	end := time.Now().UnixNano()

	raptroqLatency := uint64((end - begin) / 1000000)

	if !bytes.Equal(sourceObject, decodedObject) {
		t.Fatal("not equal")
	}

	sizeKB := packetLen / 1000
	tcpLatency := testMeasureTCPLatency(t, sourceObject)

	t.Logf("Sending %d kB packet with RaptorQ + UDP, has latency of %d ms", sizeKB, raptroqLatency)
	t.Logf("Sending %d kB packet with TCP-send has latency of %d ms", sizeKB, tcpLatency)
	t.Logf("Symbol size %d", SymbolSize)
}
