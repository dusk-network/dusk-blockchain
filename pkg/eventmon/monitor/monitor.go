package monitor

import (
	"errors"
	"net/url"

	"github.com/dusk-network/dusk-blockchain/pkg/eventmon/grpc"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	log "github.com/sirupsen/logrus"
)

var MaxAttempts int = 3
var lg = log.WithField("process", "monitor")

type Supervisor interface {
	wire.EventCollector
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
	case "grpc":
		return grpc.NewSupervisor(broker, uri), nil
	default:
		return nil, errors.New("unsupported connection type")
	}
}
