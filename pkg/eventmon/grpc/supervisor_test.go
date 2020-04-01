package grpc_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/eventmon/grpc"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-protobuf/autogen/go/monitor"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func (h *helloSrv) NotifyError(ctx context.Context, req *monitor.ErrorAlert) (*monitor.EmptyResponse, error) {
	h.requestChan <- req
	return &monitor.EmptyResponse{}, nil
}

func (h *helloSrv) NotifySlowdown(ctx context.Context, req *monitor.SlowdownAlert) (*monitor.EmptyResponse, error) {
	h.requestChan <- req
	return &monitor.EmptyResponse{}, nil
}

func TestNotifyError(t *testing.T) {
	eb := eventbus.New()
	s := grpc.NewSupervisor(eb, testUrl, 20*time.Second)
	log.AddHook(s)

	call := callTest{
		clientMethod: func() error {
			log.WithError(errors.New("this is a test")).Errorln("within grpc_test.TestNotifyError")
			return nil
		},

		tester: func(response interface{}) error {
			alert, ok := response.(*monitor.ErrorAlert)

			if !ok {
				return fmt.Errorf("unexpected request type %v", alert)
			}

			if !assert.Equal(t, monitor.Level_ERROR, alert.Level) {
				return fmt.Errorf("expected %d, instead got %d", monitor.Level_ERROR, alert.Level)
			}
			return nil
		},
	}

	Suite(t, 100, call)
	s.Halt()
}

func TestNotifySlowdown(t *testing.T) {
	eb := eventbus.New()
	s := grpc.NewSupervisor(eb, testUrl, 200*time.Millisecond)
	log.AddHook(s)
	height := uint64(200)
	testData := helper.RandomBlock(t, height, 2)
	callBlockSetup := callTest{
		clientMethod: func() error {
			return s.Client().NotifyBlockUpdate(context.Background(), *testData)
		},

		tester: emptyFunc,
	}

	callSlowdown := callTest{
		tester: func(response interface{}) error {
			alert, ok := response.(*monitor.SlowdownAlert)

			if !ok {
				return fmt.Errorf("unexpected request type %v", alert)
			}

			if !assert.Equal(t, testData.Header.Hash, alert.LastKnownHash) {
				return errors.New("wrong last known hash")
			}
			if !assert.Equal(t, testData.Header.Height, alert.LastKnownHeight) {
				return errors.New("wrong last known heigth")
			}
			return nil
		},
	}

	Suite(t, 200, callBlockSetup, callSlowdown)
	s.Halt()
}

func TestNotifySlowdownAtStart(t *testing.T) {
	eb := eventbus.New()
	s := grpc.NewSupervisor(eb, testUrl, 50*time.Millisecond)
	log.AddHook(s)

	call := callTest{
		tester: func(response interface{}) error {
			alert, ok := response.(*monitor.SlowdownAlert)

			if !ok {
				return fmt.Errorf("unexpected request type %v", alert)
			}

			if !assert.Equal(t, uint32(0), alert.TimeSinceLastBlockSec) {
				return errors.New("wrong time since last block")
			}
			if !assert.Nil(t, alert.LastKnownHash) {
				return errors.New("wrong last known hash")
			}
			if !assert.Equal(t, uint64(0), alert.LastKnownHeight) {
				return errors.New("wrong last known heigth")
			}
			return nil
		},
	}

	Suite(t, 100, call)
	s.Halt()
}
