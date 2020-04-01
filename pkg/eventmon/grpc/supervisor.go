package grpc

import (
	"context"
	"net/url"
	"sync"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-wallet/v2/block"
	log "github.com/sirupsen/logrus"
)

var lg = log.WithField("process", "monitor-client")

// Supervisor is an implementation of monitor.Supervisor interface
type Supervisor struct {
	client       *Client
	timeoutBlock time.Duration
	broker       eventbus.Broker
	stopChan     chan struct{}

	lock          sync.RWMutex
	idBlockUpdate uint32 // id of the subscription channel for the block updates
	entryChan     chan *log.Entry
	cancel        context.CancelFunc
}

// NewSupervisor returns a new instance of the Supervisor.
func NewSupervisor(broker eventbus.Broker, uri *url.URL, timeout time.Duration) *Supervisor {
	s := &Supervisor{
		client:       New(uri),
		broker:       broker,
		lock:         sync.RWMutex{},
		timeoutBlock: timeout,
		entryChan:    make(chan *log.Entry, 1),
	}

	blockChan, id := consensus.InitAcceptedBlockUpdate(broker)
	s.idBlockUpdate = id
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	go s.listenAcceptedBlock(ctx, s.entryChan, blockChan)
	return s
}

// Client is a getter for the internal grpc client. It is supposed to be used in tests only
func (s *Supervisor) Client() *Client {
	return s.client
}

// Levels as used by the log.Hook interface
func (_ *Supervisor) Levels() []log.Level {
	return []log.Level{
		log.ErrorLevel,
		log.FatalLevel,
		log.PanicLevel,
	}
}

// Fire is part of logrus.Hook interface
func (s *Supervisor) Fire(entry *log.Entry) error {
	s.entryChan <- entry
	return nil
}

// Start triggers an Hello grpc call on the underlying client
func (s *Supervisor) Start() error {
	lg.Infoln("sending start notification to server")
	return s.client.Hello()
}

// Stop halts the listening for accepted blocks and sends a Bye grpc message to
// the monitoring server
func (s *Supervisor) Stop() error {
	lg.Infoln("sending stop notification to server")
	_ = s.Halt()
	return s.client.Bye()
}

// Halt unsubscribes the supervisor from the EventBus AcceptedBlock
// notifications and stops the Slowdown alert detection routine
func (s *Supervisor) Halt() error {
	lg.Debugln("halting")
	s.cancel()
	s.broker.Unsubscribe(topics.AcceptedBlock, s.idBlockUpdate)
	return nil
}

func (s *Supervisor) listenAcceptedBlock(ctx context.Context, entryChan <-chan *log.Entry, blockChan <-chan block.Block) {
	initialTimeoutBlockAcceptance := s.timeoutBlock
	for {
		timer := time.NewTimer(s.timeoutBlock)
		lg.WithField("timeout", s.timeoutBlock.Milliseconds()).Traceln("slowdown timeout (re)created")
		select {
		case entry := <-entryChan:
			s.client.NotifyError(ctx, entry)

		case blk := <-blockChan:
			s.timeoutBlock = initialTimeoutBlockAcceptance
			timer.Stop()
			lg.WithField("timeout", s.timeoutBlock.Milliseconds()).Traceln("slowdown timeout reset")
			if err := s.client.NotifyBlockUpdate(ctx, blk); err != nil {
				lg.WithError(err).Warnln("could not send block to monitoring")
			}
		case <-timer.C:
			// relaxing the timeout
			s.timeoutBlock = s.timeoutBlock * 2
			lg.WithField("timeout", s.timeoutBlock.Milliseconds()).Traceln("doubling the slowdown timeout")
			if err := s.client.NotifyBlockSlowdown(ctx); err != nil {
				lg.WithError(err).Warnln("could not send slowdown alert to monitoring")
				continue
			}
			lg.Traceln("slowdown alert sent")
		case <-ctx.Done():
			lg.Traceln("supervisor stopped")
			return
		}
	}
}
