package rpc

import (
	"context"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-protobuf/autogen/go/phoenix"
	"google.golang.org/grpc"
)

// Client is a wrapper for a gRPC client. It establishes connection with
// the server on startup, and then handles requests from other components
// over the RPCBus.
type Client struct {
	phoenix.RuskClient
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
	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(3*time.Second))
	if err != nil {
		panic(err)
	}

	c := Client{RuskClient: phoenix.NewRuskClient(conn)}
	if err := registerMethod(rpcBus, topics.ValidateStateTransition, &c.validateSTChan); err != nil {
		panic(err)
	}
	if err := registerMethod(rpcBus, topics.ExecuteStateTransition, &c.executeSTChan); err != nil {
		panic(err)
	}

	go c.listen()
}

// re-usable function to register channels to RPCBus topics
func registerMethod(rpcBus *rpcbus.RPCBus, topic topics.Topic, c *chan rpcbus.Request) error {
	*c = make(chan rpcbus.Request, 1)
	return rpcBus.Register(topic, *c)
}

func (c Client) listen() {
	for {
		select {
		case r := <-c.validateSTChan:
			resp, err := c.ValidateStateTransition(context.Background(), &phoenix.ValidateStateTransitionRequest{Txs: r.Params.([]*phoenix.Transaction)})
			r.RespChan <- rpcbus.Response{resp, err}
			// TODO: add case for execute state transition
		}
	}
}
