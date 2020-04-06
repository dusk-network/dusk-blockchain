package monitor

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	log "github.com/sirupsen/logrus"
)

// Supervisor ...
type Supervisor interface {
	wire.EventCollector
	Reconnect() error
	Stop() error
}

// LogSupervisor ...
type LogSupervisor interface {
	Supervisor
	log.Hook
}

type mockSupervisor struct {
}

// Launch ...
func Launch(broker eventbus.Broker, monURL string) (LogSupervisor, error) {
	return &mockSupervisor{}, nil
}

func (m *mockSupervisor) Levels() []log.Level {
	return []log.Level{
		log.ErrorLevel,
		log.FatalLevel,
		log.PanicLevel,
	}
}

func (m *mockSupervisor) Fire(entry *log.Entry) error {
	return nil
}

func (m *mockSupervisor) Collect(b bytes.Buffer) error {
	return nil
}

func (m *mockSupervisor) Reconnect() error {
	return nil
}

func (m *mockSupervisor) Stop() error {
	return nil
}
