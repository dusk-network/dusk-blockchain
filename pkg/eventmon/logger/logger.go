package logger

import (
	"bytes"
	"io"
	"net"
	"runtime"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-wallet/v2/block"
	log "github.com/sirupsen/logrus"
)

const (
	// ErrWriter is the Error generated by problems with the writer
	ErrWriter byte = iota
	// ErrLog is the Error generated by problems with the Log
	ErrLog
	// ErrOther unexpected errors unrelated to logging or writer
	ErrOther
)

// LogProcessor is a TopicProcessor that intercepts messages on the gossip to create statistics and push the to the monitoring process
// It creates a new instance of logrus and writes on a io.Writer (preferrably UNIX sockets but any kind of connection will do)
type (
	LogProcessor struct {
		*log.Logger
		lastBlock         *block.Block
		p                 eventbus.Broker
		entry             *log.Entry
		acceptedBlockChan <-chan block.Block
		EntryChan         chan []byte
		quitChan          chan struct{}
		idBlockSub        uint32
	}
)

// New creates a LogProcessor
func New(p eventbus.Broker, w io.WriteCloser, formatter log.Formatter) *LogProcessor {
	logger := log.New()
	logger.Out = w
	if formatter == nil {
		logger.SetFormatter(&log.JSONFormatter{})
	}
	entry := logger.WithFields(log.Fields{
		"process": "monitor",
	})
	acceptedBlockChan, id := consensus.InitAcceptedBlockUpdate(p)
	return &LogProcessor{
		p:                 p,
		Logger:            logger,
		entry:             entry,
		acceptedBlockChan: acceptedBlockChan,
		EntryChan:         make(chan []byte, 100),
		quitChan:          make(chan struct{}, 1),
		idBlockSub:        id,
	}
}

// Listen as specified in the TopicListener interface
func (l *LogProcessor) Listen() {
	// Log number of goroutines every 5 seconds
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case entry := <-l.EntryChan:
			l.Send(entry)
		case blk := <-l.acceptedBlockChan:
			l.PublishBlockEvent(&blk)
		case <-ticker.C:
			l.logNumGoroutine()
		case <-l.quitChan:
			ticker.Stop()
			return
		}
	}
}

// LogNumGoroutine notifies the monitor of the current known number of goroutines
func (l *LogProcessor) logNumGoroutine() {
	num := runtime.NumGoroutine()
	l.entry.WithFields(log.Fields{
		"code": "goroutine",
		"nr":   num - 1,
	}).Infoln("New goroutine count")
}

// Close the listener and the Writer
func (l *LogProcessor) Close() error {
	l.p.Unsubscribe(topics.AcceptedBlock, l.idBlockSub)
	l.quitChan <- struct{}{}
	return l.Out.(io.WriteCloser).Close()
}

// Send specifies the format and the writer for the log
func (l *LogProcessor) Send(entry []byte) error {
	// Set a write deadline in case we are writing to a connection, to avoid hangs
	if conn, ok := l.Out.(net.Conn); ok {
		conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
	}

	if _, err := l.Out.Write(entry); err != nil {
		return err
	}

	// Set deadline back to zero, to avoid problems with other
	// functions that use the logger
	if conn, ok := l.Out.(net.Conn); ok {
		conn.SetWriteDeadline(time.Time{})
	}

	return nil
}

// ReportError publishes an error on the MonitorTopic
func (l *LogProcessor) ReportError(bErr byte, err error) {
	b := bytes.NewBuffer([]byte{bErr})
	msg := message.New(topics.Monitor, b)
	l.p.Publish(topics.Monitor, msg)
}

// WithTime decorates the log with time info
func (l *LogProcessor) WithTime(fields log.Fields) *log.Entry {
	entry := l.entry.WithField("time", time.Now())
	if fields == nil {
		return entry
	}
	return entry.WithFields(fields)
}

// WithError decorates the log with an error info
func (l *LogProcessor) WithError(err error) *log.Entry {
	return l.Logger.WithError(err).WithTime(time.Now())
}
