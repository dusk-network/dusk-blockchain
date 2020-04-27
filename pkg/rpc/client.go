package rpc

/*

// Client is a wrapper for a gRPC client. It establishes connection with
// the server on startup, and then handles requests from other components
// over the RPCBus.
type Client struct {
	phoenix.RuskClient
	conn           *grpc.ClientConn
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
func InitRuskClient(address string, rpcBus *rpcbus.RPCBus) *Client {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, address, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Panic(err)
	}

	c := &Client{
		RuskClient: phoenix.NewRuskClient(conn),
		conn:       conn,
	}
	if err := registerMethod(rpcBus, topics.ValidateStateTransition, &c.validateSTChan); err != nil {
		log.Panic(err)
	}
	if err := registerMethod(rpcBus, topics.ExecuteStateTransition, &c.executeSTChan); err != nil {
		log.Panic(err)
	}

	go c.listen()
	return c
}

// re-usable function to register channels to RPCBus topics
func registerMethod(rpcBus *rpcbus.RPCBus, topic topics.Topic, c *chan rpcbus.Request) error {
	*c = make(chan rpcbus.Request, 1)
	return rpcBus.Register(topic, *c)
}

func (c *Client) listen() {
	for {
		select {
		case r := <-c.validateSTChan:
			resp, err := c.ValidateStateTransition(context.Background(), &phoenix.ValidateStateTransitionRequest{Txs: r.Params.([]*phoenix.Transaction)})
			r.RespChan <- rpcbus.NewResponse(resp, err)
		case r := <-c.executeSTChan:
			// TODO: add implementation for execute state transition
			r.RespChan <- rpcbus.NewResponse(nil, nil)
		}
	}
}

// Close the connection to the gRPC server.
func (c *Client) Close() error {
	return c.conn.Close()
}
*/
