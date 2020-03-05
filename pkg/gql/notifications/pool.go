package notifications

import (
	"io"
	"net"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/gorilla/websocket"
)

// wsConn mimics the websocket.Conn from gorilla/websocket
// This abstraction allows to use mock object on unit/system testing
type wsConn interface {
	WriteMessage(messageType int, data []byte) error
	NextReader() (messageType int, r io.Reader, err error)
	WriteControl(messageType int, data []byte, deadline time.Time) error
	RemoteAddr() net.Addr
	SetWriteDeadline(t time.Time) error
	Close() error
}

// BrokerPool is a set of broker workers to provide a simple load balancing.
// Running multiple broker workers also could provide failover
type BrokerPool struct {
	ConnectionsChan chan wsConn
	QuitChan        chan bool
	workers         []*Broker
}

func NewPool(eventBus *eventbus.EventBus, brokersNum, clientsPerBroker uint) *BrokerPool {

	bp := new(BrokerPool)
	bp.workers = make([]*Broker, 0)
	bp.ConnectionsChan = make(chan wsConn, 100)

	// Instantiate all brokers
	for i := uint(0); i < brokersNum; i++ {
		br := NewBroker(i, eventBus, clientsPerBroker, bp.ConnectionsChan)
		bp.workers = append(bp.workers, br)
	}

	// Run all brokers workers
	for _, br := range bp.workers {
		go br.Run()
	}

	return bp
}

func (bp *BrokerPool) PushConn(conn *websocket.Conn) {

	if conn == nil {
		return
	}

	select {
	case bp.ConnectionsChan <- conn:
	default:
		log.Errorf("Queue is full. Discarding connection from %s", conn.RemoteAddr().String())
	}
}

func (bp *BrokerPool) Close() {

	// Closing the shared chan will trigger a cascading teardown procedure for
	// brokers and their clients.
	close(bp.ConnectionsChan)
}
