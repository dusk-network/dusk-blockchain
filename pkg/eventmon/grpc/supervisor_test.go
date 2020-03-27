package grpc_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

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
	s := grpc.NewSupervisor(eb, testUrl)
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
}

func TestNotifySlowdown(t *testing.T) {
	eb := eventbus.New()
	s := grpc.NewSupervisor(eb, testUrl)
	log.AddHook(s)
	s.SetSlowTimeout(50 * time.Millisecond)

	call := callTest{
		tester: func(response interface{}) error {
			alert, ok := response.(*monitor.SlowdownAlert)

			if !ok {
				return fmt.Errorf("unexpected request type %v", alert)
			}

			if !assert.Equal(t, uint32(0), alert.TimeSinceLastBlock) {
				return errors.New("wrong time since last block")
			}
			return nil
		},
	}

	Suite(t, 100, call)
}
