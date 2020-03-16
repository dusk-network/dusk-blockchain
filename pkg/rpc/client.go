package rpc

import (
	"context"

	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
)

type Client struct {
	rusk.RuskClient
	validateSTChan chan rpcbus.Request
	executeSTChan  chan rpcbus.Request
}

// InitRuskClient opens the connection with the Rusk gRPC server, and
// launches a goroutine which listens for RPCBus calls that concern
// the Rusk server.
//
// As the Rusk server is a fundamental part of the node functionality,
// this function will panic if the connection can not be established
// successfully.
func InitRuskClient(address string, rpcBus *rpcbus.RPCBus) {
	conn, err := grpc.Dial(address)
	if err != nil {
		panic(err)
	}

	c := Client{RuskClient: rusk.NewRuskClient(conn)}
	if err := registerMethod(rpcBus, topics.ValidateStateTransition, c.validateSTChan); err != nil {
		panic(err)
	}
	if err := registerMethod(rpcBus, topics.ExecuteStateTransition, c.executeSTChan); err != nil {
		panic(err)
	}

	go c.listen()
}

func registerMethod(rpcBus *rpcbus.RPCBus, topic topics.Topic, c chan rpcbus.Request) error {
	c = make(chan rpcbus.Request, 1)
	return rpcBus.Register(topic, c)
}

func (c Client) listen() {
	for {
		select {
		case r := <-c.validateSTChan:
			resp, err := c.ValidateStateTransition(context.Background(), &rusk.ValidateStateTransitionRequest{Txs: r.Params.([]transactions.Transaction)})
			r.RespChan <- rpcbus.Response{resp, err}
		case r := <-c.executeSTChan:
			resp, err := c.ExecuteStateTransition(context.Background(), &rusk.ExecuteStateTransition{Txs: r.Params.([]transactions.Transaction)})
			r.RespChan <- rpcbus.Response{resp, err}
		}
	}
}
