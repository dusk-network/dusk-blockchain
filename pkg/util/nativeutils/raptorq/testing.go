// +build

package raptorq

import (
	"bytes"
	"io"
	"net"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
)

// tcpreader is a simple tcp listner that reads a single packet from a client conn
// useful for benchmark purposes only
type tcpreader struct {
	listener *net.TCPListener

	length int
	done   chan bool

	receivedPacket []byte
}

func newTCPReader(lAddr *net.TCPAddr, length int, done chan bool) *tcpreader {

	l, err := net.ListenTCP("tcp4", lAddr)
	if err != nil {
		log.Panic(err)
	}

	reader := &tcpreader{
		listener: l,
		length:   length,
		done:     done,
	}

	return reader
}

// Serve starts accepting and processing TCP connection and packets
func (r *tcpreader) Serve() {

	for {
		conn, err := r.listener.AcceptTCP()
		if err != nil {
			log.WithError(err).Warn("Error on tcp accept")
			return
		}

		// processPacket as for now a single-packet-per-connection is allowed
		go r.processPacket(conn)
	}
}

func (r *tcpreader) processPacket(conn *net.TCPConn) {

	// Current impl expects only one TCPFrame per connection
	defer func() {
		if conn != nil {
			_ = conn.Close()
		}
	}()

	// Read frame payload Set a new deadline for the connection.
	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))

	// Read packet payload
	payload := make([]byte, r.length)
	if _, err := io.ReadFull(conn, payload); err != nil {
		panic(err)
	}

	r.receivedPacket = payload
	r.done <- true
}

func tcpSendAndClose(conn *net.TCPConn, data []byte) {

	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))

	// Write data stream
	if _, err := conn.Write(data); err != nil {
		panic(err)
	}

	_ = conn.Close()
}

// testMeasureLatency measure latency of tcp sending in dial-write-close approach
// NB: Benchmark is intended to be used with following env changes:
// ip link set dev lo mtu 1500
// tc qdisc add dev lo root netem delay 10ms 15ms 15% reorder 25% 50% corrupt 5% 10%
func testMeasureTCPLatency(t *testing.T, sourceObject []byte) uint64 {

	raddr, err := net.ResolveTCPAddr("tcp4", "127.0.0.1:44444")
	if err != nil {
		t.Error(err)
	}

	done := make(chan bool)

	// Start remote service
	r := newTCPReader(raddr, len(sourceObject), done)

	go r.Serve()

	time.Sleep(1 * time.Second)

	// Send a single sourceObject
	conn, err := net.DialTCP("tcp4", nil, raddr)
	if err != nil {
		panic(err)
	}

	begin := time.Now().UnixNano()

	go tcpSendAndClose(conn, sourceObject)

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("tcp timeout error")
	}

	latency := uint64((time.Now().UnixNano() - begin) / 1000000)

	if !bytes.Equal(sourceObject, r.receivedPacket) {
		t.Fatal("not equal")
	}

	return latency
}
