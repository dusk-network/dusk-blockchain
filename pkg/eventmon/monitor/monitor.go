package monitor

import (
	"bytes"
	"errors"
	"io"
	"net"
	"net/url"
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/eventmon/logger"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	log "github.com/sirupsen/logrus"
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

func Launch(broker eventbus.Broker, monUrl string) (LogSupervisor, error) {

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
	broker     eventbus.Broker
	lock       sync.Mutex
	processor  *logger.LogProcessor
	uri        *url.URL
	attempts   int
	activeProc bool
}

func (m *unixSupervisor) Levels() []log.Level {
	return []log.Level{
		log.ErrorLevel,
		log.FatalLevel,
		log.PanicLevel,
	}
}

func (m *unixSupervisor) Fire(entry *log.Entry) error {
	if m.activeProc {
		// Format the entry first. Since the logger still uses this entry after the hook has fired,
		// race conditions can occur if the `Logger` formats the entry after receiving it through the channel.
		// So, we format it here and send the bytes over instead.
		formatted, err := m.processor.Logger.Formatter.Format(entry)
		if err != nil {
			return err
		}

		// Drop events if the queue is filled up, to avoid extended lockups
		select {
		case m.processor.EntryChan <- formatted:
		default:
		}
	}
	return nil
}

func newUnixSupervisor(broker eventbus.Broker, uri *url.URL) (LogSupervisor, error) {

	logProc, err := initLogProcessor(broker, uri)
	if err != nil {
		return nil, err
	}

	return &unixSupervisor{
		broker:     broker,
		lock:       sync.Mutex{},
		processor:  logProc,
		uri:        uri,
		activeProc: true,
	}, nil
}

func (m *unixSupervisor) Collect(b bytes.Buffer) error {
	// TODO: maybe diversify the action depending on the errors in the future
	var err error
	// whatever the case, we are going to reset the supervisor
	defer m.resetAttempts()

	err = deserializeError(&b)
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

	proc, err := initLogProcessor(m.broker, m.uri)
	if err != nil {
		return err
	}
	go proc.Listen()
	m.activeProc = true
	m.processor = proc
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
		return m.processor.Close()
	}
	return nil
}

func initLogProcessor(broker eventbus.Broker, uri *url.URL) (*logger.LogProcessor, error) {
	wc, err := start(uri)
	if err != nil {
		return nil, err
	}

	logProcessor := logger.New(broker, wc, nil)
	go logProcessor.Listen()

	return logProcessor, nil
}

func start(uri *url.URL) (io.WriteCloser, error) {
	conn, err := net.Dial(uri.Scheme, uri.Path)
	if err != nil {
		return nil, err
	}

	return conn, nil
}
