package grpc_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/eventmon/grpc"
	"github.com/dusk-network/dusk-protobuf/autogen/go/monitor"
	"github.com/stretchr/testify/assert"
)

//NotifyBlock is the server GRPC method for receiving the block
func (h *helloSrv) NotifyBlock(ctx context.Context, req *monitor.BlockUpdate) (*monitor.EmptyResponse, error) {
	h.requestChan <- req
	return &monitor.EmptyResponse{}, nil
}

// TestNotifyBlockUpdate tests the correct sending and reception of a
// BlockUpdate
func TestNotifyBlockUpdate(t *testing.T) {
	client := grpc.New(testUrl)

	height := uint64(200)
	testData := helper.RandomBlock(t, height, 2)
	call := callTest{
		clientMethod: func() error {
			return client.NotifyBlockUpdate(context.Background(), *testData)
		},

		tester: func(response interface{}) error {
			update, ok := response.(*monitor.BlockUpdate)

			if !ok {
				return fmt.Errorf("unexpected request type %v", update)
			}

			if !assert.Equal(t, testData.Header.Height, update.Height) { // should be 200
				return errors.New("wrong height")
			}
			if !assert.Equal(t, testData.Header.Hash, update.Hash) {
				return errors.New("wrong hash")
			}

			if !assert.Equal(t, testData.Header.Timestamp, update.Timestamp) {
				return errors.New("wrong timestamp")
			}

			if !assert.Equal(t, uint32(len(testData.Txs)), update.TxAmount) {
				return errors.New("wrong tx amount")
			}
			if !assert.Equal(t, uint32(0), update.BlockTimeSec) { // since there was no block before this one, the time difference should not have been communicated
				return errors.New("block time should be 0")
			}
			return nil
		},
	}
	Suite(t, 100, call)
}

// TestBlockTime tests the correct calculation and reception of the blocktime
func TestBlockTime(t *testing.T) {
	client := grpc.New(testUrl)

	height := uint64(200)
	firstBlock := helper.RandomBlock(t, height, 2)
	secondBlock := helper.RandomBlock(t, height+1, 2)

	// first we submit a block
	firstCall := callTest{
		// sending first block
		clientMethod: func() error {
			return client.NotifyBlockUpdate(context.Background(), *firstBlock)
		},

		// nothing to test here
		tester: emptyFunc,
	}

	// we submit another block
	secondCall := callTest{
		clientMethod: func() error {
			// adding 2 seconds to test block time
			secondBlock.Header.Timestamp = firstBlock.Header.Timestamp + 2
			return client.NotifyBlockUpdate(context.Background(), *secondBlock)
		},

		tester: func(response interface{}) error {
			update, ok := response.(*monitor.BlockUpdate)

			if !ok {
				return fmt.Errorf("unexpected request type %v", update)
			}

			if !assert.Equal(t, uint32(2), update.BlockTimeSec) { // since there was no block before this one, the time difference should not have been communicated
				return errors.New("block time should be 2")
			}
			return nil
		},
	}

	Suite(t, 100, firstCall, secondCall)
}
