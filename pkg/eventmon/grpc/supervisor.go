package grpc

import (
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
}

// NewSupervisor returns a new instance of the Supervisor.
func NewSupervisor(broker eventbus.Broker, uri *url.URL, timeout time.Duration) *Supervisor {
	s := &Supervisor{
		client:       New(uri),
		broker:       broker,
		stopChan:     make(chan struct{}, 1),
		lock:         sync.RWMutex{},
		timeoutBlock: timeout,
	}

	blockChan, id := consensus.InitAcceptedBlockUpdate(broker)
	s.idBlockUpdate = id
	go s.listenAcceptedBlock(blockChan)
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
	return s.client.NotifyError(entry)
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
	s.broker.Unsubscribe(topics.AcceptedBlock, s.idBlockUpdate)
	s.stopChan <- struct{}{}
	return nil
}

func (s *Supervisor) listenAcceptedBlock(blockChan <-chan block.Block) {
	initialTimeoutBlockAcceptance := s.timeoutBlock
	for {
		timer := time.NewTimer(s.timeoutBlock)
		lg.WithField("timeout", s.timeoutBlock.Milliseconds()).Traceln("slowdown timeout (re)created")
		select {
		case blk := <-blockChan:
			s.timeoutBlock = initialTimeoutBlockAcceptance
			timer.Stop()
			lg.WithField("timeout", s.timeoutBlock.Milliseconds()).Traceln("slowdown timeout reset")
			if err := s.client.NotifyBlockUpdate(blk); err != nil {
				lg.WithError(err).Warnln("could not send block to monitoring")
			}
		case <-timer.C:
			// relaxing the timeout
			s.timeoutBlock = s.timeoutBlock * 2
			lg.WithField("timeout", s.timeoutBlock.Milliseconds()).Traceln("doubling the slowdown timeout")
			if err := s.client.NotifyBlockSlowdown(); err != nil {
				lg.WithError(err).Warnln("could not send slowdown alert to monitoring")
				continue
			}
			lg.Traceln("slowdown alert sent")
		case <-s.stopChan:
			lg.Traceln("supervisor stopped")
			return
		}
	}
}
