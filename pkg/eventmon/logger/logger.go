package logger

import (
	"bytes"
	"io"
	"io/ioutil"
	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/agreement"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

// Make sure LogProcessor implements the logrus.Hook interface
var _ log.Hook = (*LogProcessor)(nil)

// Make sure LogProcessor implements the TopicProcessor interface
var _ wire.TopicProcessor = (*LogProcessor)(nil)

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
		lastInfo *blockInfo
		p        wire.EventPublisher
		active   bool
		entry    *log.Entry
	}

	blockInfo struct {
		t time.Time
		*agreement.Agreement
	}
)

func New(p wire.EventPublisher, w io.WriteCloser, formatter log.Formatter) *LogProcessor {
	logger := log.New()
	logger.Out = w
	if formatter == nil {
		logger.SetFormatter(&log.JSONFormatter{})
	}
	entry := logger.WithFields(log.Fields{
		"process": "monitor",
	})
	return &LogProcessor{
		p:      p,
		Logger: logger,
		active: true,
		entry:  entry,
	}
}

func (l *LogProcessor) Wire(w io.WriteCloser) {
	l.active = true
	_ = l.Close()
	l.Out = w
}

func (l *LogProcessor) Close() error {
	if !l.active {
		return nil
	}

	l.active = false
	return l.Out.(io.WriteCloser).Close()
}

func (l *LogProcessor) Levels() []log.Level {
	return []log.Level{
		// log.WarnLevel,
		log.ErrorLevel,
		log.FatalLevel,
		log.PanicLevel,
	}
}

func (l *LogProcessor) Fire(entry *log.Entry) error {
	formatted, err := l.Formatter.Format(entry)
	if err != nil {
		return err
	}

	if _, err = l.Out.Write(formatted); err != nil {
		return err
	}

	return nil
}

// Process creates a copy of the message, checks the topic header
func (l *LogProcessor) Process(buf *bytes.Buffer) (*bytes.Buffer, error) {
	var newBuf bytes.Buffer
	r := io.TeeReader(buf, &newBuf)

	topic, err := topics.Extract(r)
	if err != nil {
		return nil, err
	}

	aggro, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	if topic == topics.Agreement {
		go l.PublishRoundEvent(aggro)
	}

	return &newBuf, nil
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
