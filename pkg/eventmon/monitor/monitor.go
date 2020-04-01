package monitor

import (
	"errors"
	"net/url"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/eventmon/grpc"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	log "github.com/sirupsen/logrus"
)

type Supervisor interface {
	log.Hook
	Start() error
	Stop() error
}

type SupervisorCollector interface {
	wire.EventCollector
	Supervisor
}

func Launch(broker eventbus.Broker, blockTimeThreshold time.Duration, monType, monUrl string) (Supervisor, error) {
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
