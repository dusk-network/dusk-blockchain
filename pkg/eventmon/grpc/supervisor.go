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

var lg = log.WithField("process", "grpc-supervisor")

type Supervisor struct {
	Client       *Client
	timeoutBlock time.Duration
	broker       eventbus.Broker
	stopChan     chan struct{}

	lock          sync.RWMutex
	idBlockUpdate uint32 // id of the subscription channel for the block updates
}

func NewSupervisor(broker eventbus.Broker, uri *url.URL, timeout time.Duration) *Supervisor {
	c := New(uri)
	s := &Supervisor{
		Client:       c,
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

func (_ *Supervisor) Levels() []log.Level {
	return []log.Level{
		log.ErrorLevel,
		log.FatalLevel,
		log.PanicLevel,
	}
}

func (s *Supervisor) Fire(entry *log.Entry) error {
	return s.Client.NotifyError(entry)
}

func (s *Supervisor) Stop() error {
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
			if err := s.Client.NotifyBlockUpdate(blk); err != nil {
				lg.WithError(err).Warnln("could not send block to monitoring")
			}
		case <-timer.C:
			// relaxing the timeout
			s.timeoutBlock = s.timeoutBlock * 2
			lg.WithField("timeout", s.timeoutBlock.Milliseconds()).Traceln("doubling the slowdown timeout")
			if err := s.Client.NotifyBlockSlowdown(); err != nil {
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
