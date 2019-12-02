package requesting

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/peermsg"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
)

type AgreementCache struct {
	lock        sync.RWMutex
	agreements  []bytes.Buffer
	latestRound uint64

	responseChan chan<- *bytes.Buffer
}

func NewAgreementCache(sub eventbus.Subscriber, responseChan chan<- *bytes.Buffer) *AgreementCache {
	a := &AgreementCache{
		agreements:   make([]bytes.Buffer, 0, 4),
		responseChan: responseChan,
	}

	sub.Subscribe(topics.AddAgreements, eventbus.NewCallbackListener(a.addAgreements))
	return a
}

func (a *AgreementCache) addAgreements(b bytes.Buffer) error {
	a.lock.Lock()
	defer a.lock.Unlock()
	var round uint64
	if err := encoding.ReadUint64LE(&b, &round); err != nil {
		return err
	}

	a.latestRound = round
	// Prepend agreements to slice
	a.agreements = append([]bytes.Buffer{b}, a.agreements...)
	if len(a.agreements) > 3 {
		a.agreements = a.agreements[:3]
	}

	return nil
}

func (a *AgreementCache) SendAgreements(m *bytes.Buffer) error {
	a.lock.RLock()
	defer a.lock.RUnlock()
	msg := &peermsg.GetAgreements{}
	if err := msg.Decode(m); err != nil {
		return err
	}

	if msg.Round > a.latestRound || msg.Round < a.latestRound-2 {
		return fmt.Errorf("no agreement messages for round %v\n", msg.Round)
	}

	// Send agreements
	idx := a.latestRound - msg.Round
	a.responseChan <- &a.agreements[idx]
	return nil
}
