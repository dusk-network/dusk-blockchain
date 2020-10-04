package firststep

import (
	"bytes"
	"errors"
	"sync"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/reduction"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
)

// Helper for reducing test boilerplate
type Helper struct {
	*reduction.Helper
	lock               sync.RWMutex
	failOnFetching     bool
	failOnVerification bool
}

// NewHelper creates a Helper used for testing the first step Reducer.
// The constructor also launches goroutines
// that intercept RPC calls made by the first step Reducer.
func NewHelper(provisioners int, timeOut time.Duration) *Helper {
	hlp := &Helper{
		Helper:             reduction.NewHelper(provisioners, timeOut),
		failOnFetching:     false,
		failOnVerification: false,
	}

	go hlp.provideCandidateBlock()
	go hlp.processCandidateVerificationRequest()
	return hlp
}

// FailOnVerification tells the RPC bus to return an error
func (hlp *Helper) FailOnVerification(flag bool) {
	hlp.lock.Lock()
	defer hlp.lock.Unlock()
	hlp.failOnVerification = flag
}

// FailOnFetching sets the failOnFetching flag
func (hlp *Helper) FailOnFetching(flag bool) {
	hlp.lock.Lock()
	defer hlp.lock.Unlock()
	hlp.failOnFetching = flag
}

func (hlp *Helper) shouldFailFetching() bool {
	hlp.lock.RLock()
	defer hlp.lock.RUnlock()
	f := hlp.failOnFetching
	return f
}

func (hlp *Helper) shouldFailVerification() bool {
	hlp.lock.RLock()
	defer hlp.lock.RUnlock()
	f := hlp.failOnVerification
	return f
}

func (hlp *Helper) provideCandidateBlock() {
	c := make(chan rpcbus.Request, 1)
	_ = hlp.RPCBus.Register(topics.GetCandidate, c)
	for {
		r := <-c
		if hlp.shouldFailFetching() {
			r.RespChan <- rpcbus.NewResponse(bytes.Buffer{}, errors.New("could not get candidate block"))
			continue
		}

		r.RespChan <- rpcbus.NewResponse(message.Candidate{}, nil)
	}
}

func (hlp *Helper) processCandidateVerificationRequest() {
	v := make(chan rpcbus.Request, 1)
	if err := hlp.RPCBus.Register(topics.VerifyCandidateBlock, v); err != nil {
		panic(err)
	}
	for {
		r := <-v
		if hlp.shouldFailVerification() {
			r.RespChan <- rpcbus.NewResponse(nil, errors.New("verification failed"))
			continue
		}

		r.RespChan <- rpcbus.NewResponse(nil, nil)
	}
}
