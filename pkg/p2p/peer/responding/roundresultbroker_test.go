package responding_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/peer/responding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/stretchr/testify/assert"
)

const successStr = "success"

// Test the functionality of the RoundResultBroker.
func TestRoundResultBroker(t *testing.T) {
	rb := rpcbus.New()
	respChan := make(chan *bytes.Buffer, 1)
	r := responding.NewRoundResultBroker(rb, respChan)
	correctRound := uint64(5)
	quitChan := provideRoundResult(rb, correctRound)

	// First, ask for a round result which we don't have
	buf := new(bytes.Buffer)
	assert.NoError(t, encoding.WriteUint64LE(buf, 1))
	assert.Error(t, r.ProvideRoundResult(buf))

	// Now, ask for the correct one
	buf = new(bytes.Buffer)
	assert.NoError(t, encoding.WriteUint64LE(buf, correctRound))
	assert.NoError(t, r.ProvideRoundResult(buf))

	// Should have `successStr` in `respChan`
	result := <-respChan
	// First byte of result.String should be a topic, so we remove it
	// for the check
	assert.Equal(t, successStr, result.String()[1:])

	// Clean up goroutine
	quitChan <- struct{}{}
}

func provideRoundResult(rb *rpcbus.RPCBus, correctRound uint64) chan struct{} {
	quitChan := make(chan struct{}, 1)
	reqChan := make(chan rpcbus.Request, 1)

	if err := rb.Register(rpcbus.GetRoundResults, reqChan); err != nil {
		panic(err)
	}

	go func(reqChan chan rpcbus.Request, quitChan chan struct{}, correctRound uint64) {
		for {
			select {
			case r := <-reqChan:
				var round uint64
				if err := encoding.ReadUint64LE(&r.Params, &round); err != nil {
					panic(err)
				}

				if round == correctRound {
					r.RespChan <- rpcbus.Response{*bytes.NewBufferString(successStr), nil}
					continue
				}

				r.RespChan <- rpcbus.Response{bytes.Buffer{}, errors.New("not found")}
			case <-quitChan:
				return
			}
		}
	}(reqChan, quitChan, correctRound)

	return quitChan
}
