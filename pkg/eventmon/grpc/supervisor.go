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

var defaultTimeout = 20 * time.Second

type Supervisor struct {
	timeoutBlock time.Duration
	broker       eventbus.Broker
	client       *Client

	lock          sync.RWMutex
	idBlockUpdate uint32 // id of the subscription channel for the block updates
}

func NewSupervisor(broker eventbus.Broker, uri *url.URL) *Supervisor {
	c := New(uri)
	s := &Supervisor{
		broker:       broker,
		lock:         sync.RWMutex{},
		client:       c,
		timeoutBlock: defaultTimeout,
	}

	blockChan, id := consensus.InitAcceptedBlockUpdate(broker)
	s.idBlockUpdate = id
	go s.listenAcceptedBlock(blockChan)

	return s
}

func (s *Supervisor) SlowTimeout() time.Duration {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.timeoutBlock
}

func (s *Supervisor) SetSlowTimeout(t time.Duration) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.timeoutBlock = t
}

func (s *Supervisor) Hello() error {
	return s.client.Hello()
}

func (_ *Supervisor) Levels() []log.Level {
	return []log.Level{
		log.ErrorLevel,
		log.FatalLevel,
		log.PanicLevel,
	}
}

func (s *Supervisor) Fire(entry *log.Entry) error {
	return s.client.NotifyError(entry)
}

func (s *Supervisor) Stop() error {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.broker.Unsubscribe(topics.AcceptedBlock, s.idBlockUpdate)

	// TODO: close the client streams if any
	return s.client.Bye()
}

func (s *Supervisor) listenAcceptedBlock(blockChan <-chan block.Block) {
	initialTimeoutBlockAcceptance := s.SlowTimeout()
	for {
		timer := time.NewTimer(s.SlowTimeout())
		select {
		case blk := <-blockChan:
			s.SetSlowTimeout(initialTimeoutBlockAcceptance)
			timer.Stop()
			// get block
			// send block
			if err := s.client.NotifyBlockUpdate(blk); err != nil {
				log.WithError(err).Warnln("could not send block to monitoring")
			}
		case <-timer.C:
			// relaxing the timeout so
			currentTimeout := s.SlowTimeout()
			s.SetSlowTimeout(currentTimeout * 2)
			if err := s.client.NotifyBlockSlowdown(); err != nil {
				log.WithError(err).Warnln("could not send slowdown alert to monitoring")
			}
		}
	}
}
