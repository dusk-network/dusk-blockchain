package monitor

import (
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/eventmon/logger"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

func LaunchLogMonitor(subscriber wire.EventSubscriber, w io.WriteCloser) {
	logProcessor := logger.New(w, nil)
	subscriber.RegisterPreprocessor(string(topics.Gossip), logProcessor)
}
