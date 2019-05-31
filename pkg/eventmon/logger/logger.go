package logger

import (
	"bytes"
	"io"
	"io/ioutil"
	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/agreement"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

// LogProcessor is a TopicProcessor that intercepts messages on the gossip to create statistics and push the to the monitoring process
// It creates a new instance of logrus and writes on a io.Writer (preferrably UNIX sockets but any kind of connection will do)
type LogProcessor struct {
	*log.Entry
	lastInfo *blockInfo
}

type blockInfo struct {
	time.Time
	*agreement.Agreement
}

func New(w io.Writer, formatter log.Formatter) *LogProcessor {
	logger := log.New()
	logger.Out = w
	if formatter == nil {
		logger.SetFormatter(&log.JSONFormatter{})
	}
	entry := logger.WithFields(log.Fields{
		"process": "monitor",
	})
	return &LogProcessor{Entry: entry}
}

func (l *LogProcessor) Process(buf *bytes.Buffer) (*bytes.Buffer, error) {
	var newBuf bytes.Buffer
	r := io.TeeReader(buf, &newBuf)

	topic, err := topics.Extract(r)
	if err != nil {
		return nil, err
	}

	if topic == topics.Agreement {
		aggro, err := ioutil.ReadAll(r)
		if err != nil {
			return nil, err
		}

		go l.PublishRoundEvent(aggro)
	}

	return &newBuf, nil
}

func (l *LogProcessor) WithTime(fields log.Fields) *log.Entry {
	entry := l.WithField("time", time.Now())
	if fields == nil {
		return entry
	}
	return entry.WithFields(fields)
}

func (l *LogProcessor) WithError(err error) *log.Entry {
	return l.Logger.WithError(err).WithTime(time.Now())
}
