package monitor

import (
	"bytes"
	"errors"
	"io"
	"net"
	"net/url"
	"sync"

	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/eventmon/logger"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

var MaxAttempts int = 3
var lg = log.WithField("process", "monitor")

type Supervisor interface {
	wire.EventCollector
	Reconnect() error
	Stop() error
}

type LogSupervisor interface {
	Supervisor
	log.Hook
}

func Launch(broker wire.EventBroker, monUrl string) (LogSupervisor, error) {

	uri, err := url.Parse(monUrl)
	if err != nil {
		return nil, err
	}
	switch uri.Scheme {
	case "file":
		return nil, errors.New("file dumping on the logger is not implemented right now")
	case "unix":
		return newUnixSupervisor(broker, uri)
	default:
		return nil, errors.New("unsupported connection type")
	}
}

type unixSupervisor struct {
	broker      wire.EventBroker
	lock        sync.Mutex
	processor   *logger.LogProcessor
	uri         *url.URL
	processorId uint32
	attempts    int
	activeProc  bool
}

func (m *unixSupervisor) Levels() []log.Level {
	return []log.Level{
		// log.WarnLevel,
		log.ErrorLevel,
		log.FatalLevel,
		log.PanicLevel,
	}
}

func (m *unixSupervisor) Fire(entry *log.Entry) error {
	if m.activeProc {
		return m.processor.Send(entry)
	}
	return nil
}

func newUnixSupervisor(broker wire.EventBroker, uri *url.URL) (LogSupervisor, error) {

	logProc, id, err := initLogProcessor(broker, uri)
	if err != nil {
		return nil, err
	}

	return &unixSupervisor{
		broker:      broker,
		lock:        sync.Mutex{},
		processor:   logProc,
		uri:         uri,
		processorId: id,
		activeProc:  true,
	}, nil
}

func (m *unixSupervisor) Collect(b *bytes.Buffer) error {
	// TODO: maybe diversify the action depending on the errors in the future
	var err error
	// whatever the case, we are going to reset the supervisor
	defer m.resetAttempts()

	err = deserializeError(b)
	lg.WithField("op", "Collect").WithError(err).Warnln("Error notified by the LogProcessor. Attempting to reconnect to the monitoring server")

	m.attempts++
	for ; m.attempts < MaxAttempts; m.attempts++ {
		err = m.Reconnect()
		if err == nil {
			break
		}
		lg.WithField("attempt", m.attempts).WithError(err).Warnln("Reconnecting to the monitor failed")
	}

	if m.attempts > MaxAttempts {
		//giving up
		_ = m.Stop()
		lg.WithError(err).Errorln("cannot reconnect to the monitoring system. Giving up")
		return err
	}

	lg.WithField("attempt", m.attempts).Infoln("Successfully reconnected to the monitoring server")
	return nil
}

func (m *unixSupervisor) resetAttempts() {
	m.attempts = 0
}

func deserializeError(b *bytes.Buffer) error {
	bErr := make([]byte, 1)
	if _, err := b.Read(bErr); err != nil {
		return err
	}

	switch bErr[0] {
	case logger.ErrWriter:
		return errors.New("Connection error")
	case logger.ErrLog:
		return errors.New("Log/monitoring error")
	case logger.ErrOther:
	default:
		return errors.New("Unspecified error")
	}
	return nil
}

func (m *unixSupervisor) Reconnect() error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if err := m.stop(); err != nil {
		return err
	}

	proc, id, err := initLogProcessor(m.broker, m.uri)
	if err != nil {
		return err
	}
	m.activeProc = true
	m.processor = proc
	m.processorId = id
	m.attempts = 0
	return nil
}

func (m *unixSupervisor) Stop() error {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.stop()
}

func (m *unixSupervisor) stop() error {
	if m.activeProc {
		m.activeProc = false
		m.broker.RemovePreprocessor(string(topics.Gossip), m.processorId)
		return m.processor.Close()
	}
	return nil
}

func initLogProcessor(broker wire.EventBroker, uri *url.URL) (*logger.LogProcessor, uint32, error) {
	wc, err := start(uri)
	if err != nil {
		return nil, uint32(0), err
	}

	logProcessor := logger.New(broker, wc, nil)
	ids := broker.RegisterPreprocessor(string(topics.Gossip), logProcessor)
	go logProcessor.LogNumGoroutine()

	return logProcessor, ids[0], nil
}

func start(uri *url.URL) (io.WriteCloser, error) {
	conn, err := net.Dial(uri.Scheme, uri.Path)
	if err != nil {
		return nil, err
	}

	return conn, nil
}
