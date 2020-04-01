package monitor

import (
	"errors"
	"net/url"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/eventmon/grpc"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	log "github.com/sirupsen/logrus"
)

// Supervisor is a monitor client that pushes data and alerts to a server. It
// is also a logrus Hook to facilitate capturing error logs when emitted
type Supervisor interface {
	log.Hook
	Start() error
	Stop() error
}

// New is a factory method to create a Supervisor
func New(broker eventbus.Broker, blockTimeThreshold time.Duration, monType, monUrl string) (Supervisor, error) {
	uri, err := url.Parse(monUrl)
	if err != nil {
		return nil, err
	}
	switch monType {
	case "grpc":
		return grpc.NewSupervisor(broker, uri, blockTimeThreshold), nil
	default:
		return nil, errors.New("unsupported monitor type")
	}
}
