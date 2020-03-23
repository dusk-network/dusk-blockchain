package grpc_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/eventmon/grpc"
	"github.com/dusk-network/dusk-protobuf/autogen/go/monitor"
	"github.com/stretchr/testify/assert"
)

func (h *helloSrv) NotifyBlock(ctx context.Context, req *monitor.BlockUpdate) (*monitor.EmptyResponse, error) {
	h.requestChan <- req
	return &monitor.EmptyResponse{}, nil
}

func TestNotifyBlockUpdate(t *testing.T) {
	client := grpc.New("", "7878")

	height := uint64(200)
	testData := helper.RandomBlock(t, height, 2)
	method := func() error {
		return client.NotifyBlockUpdate(testData)
	}

	now := time.Now()
	testStartSec := now.Unix()
	tester := func(response interface{}) error {
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

		stamp, err := time.Parse(time.RFC3339, update.Timestamp)
		if !assert.NoError(t, err) {
			return err
		}

		sentSec := stamp.Unix()
		// asserting that the timestamp of (sending) the block is received in
		// good order (aka, is approximately the time calculated immediately
		// before sending the block
		if !assert.InDelta(t, sentSec, testStartSec, 1) {
			return errors.New("wrong timestamp")
		}

		if !assert.Equal(t, uint32(len(testData.Txs)), update.TxAmount) {
			return errors.New("wrong tx amount")
		}
		if !assert.Equal(t, uint32(0), update.BlockTimeSec) { // since there was no block before this one, the time difference should not have been communicated
			return errors.New("block time should be nil")
		}
		return nil
	}
	Suite(t, method, tester)
}
