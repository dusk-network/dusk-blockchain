package grpc

import (
	"context"
	"net"
	"time"

	pb "github.com/dusk-network/dusk-protobuf/autogen/go/monitor"
	"github.com/dusk-network/dusk-wallet/v2/block"
	g "google.golang.org/grpc"
)

type Client struct {
	tgt       string
	lastBlock *block.Block
}

// New creates a Client
// TODO: add TLS certificates
func New(host, port string) *Client {
	addr := net.JoinHostPort(host, port)
	return &Client{addr, nil}
}

func (c *Client) sendOnce(callback func(pb.MonitorClient, context.Context) error) error {
	conn, err := g.Dial(c.tgt, g.WithInsecure(), g.WithBlock())
	if err != nil {
		return err
	}

	defer conn.Close()
	mon := pb.NewMonitorClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)

	defer cancel()
	return callback(mon, ctx)
}
