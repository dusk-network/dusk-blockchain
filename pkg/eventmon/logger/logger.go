package logger

import (
	"bytes"
	"io"
	"runtime"
	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

const MonitorTopic = "monitor_topic"

const (
	ErrWriter byte = iota
	ErrLog
	ErrOther
)

// LogProcessor is a TopicProcessor that intercepts messages on the gossip to create statistics and push the to the monitoring process
// It creates a new instance of logrus and writes on a io.Writer (preferrably UNIX sockets but any kind of connection will do)
type (
	LogProcessor struct {
		*log.Logger
		lastBlock         *block.Block
		p                 wire.EventPublisher
		entry             *log.Entry
		acceptedBlockChan <-chan block.Block
		listener          *wire.TopicListener
	}
)

func New(p wire.EventBroker, w io.WriteCloser, formatter log.Formatter) *LogProcessor {
	logger := log.New()
	logger.Out = w
	if formatter == nil {
		logger.SetFormatter(&log.JSONFormatter{})
	}
	entry := logger.WithFields(log.Fields{
		"process": "monitor",
	})
	acceptedBlockChan, listener := consensus.InitAcceptedBlockUpdate(p)
	return &LogProcessor{
		p:                 p,
		Logger:            logger,
		entry:             entry,
		acceptedBlockChan: acceptedBlockChan,
		listener:          listener,
	}
}

func (l *LogProcessor) ListenForNewBlocks() {
	for {
		blk := <-l.acceptedBlockChan
		l.PublishBlockEvent(&blk)
	}
}

func (l *LogProcessor) LogNumGoroutine() {
	for {
		time.Sleep(5 * time.Second)
		num := runtime.NumGoroutine()
		l.entry.WithFields(log.Fields{
			"code": "goroutine",
			"nr":   num - 1,
		}).Infoln("New goroutine count")
	}
}
func (l *LogProcessor) Close() error {
	l.listener.Quit()
	return l.Out.(io.WriteCloser).Close()
}

func (l *LogProcessor) Send(entry *log.Entry) error {
	formatted, err := l.Formatter.Format(entry)
	if err != nil {
		return err
	}

	if _, err = l.Out.Write(formatted); err != nil {
		return err
	}

	return nil
}

func (l *LogProcessor) ReportError(bErr byte, err error) {
	b := bytes.NewBuffer([]byte{bErr})
	l.p.Publish(MonitorTopic, b)
}

func (l *LogProcessor) WithTime(fields log.Fields) *log.Entry {
	entry := l.entry.WithField("time", time.Now())
	if fields == nil {
		return entry
	}
	return entry.WithFields(fields)
}

func (l *LogProcessor) WithError(err error) *log.Entry {
	return l.Logger.WithError(err).WithTime(time.Now())
}
