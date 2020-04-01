package grpc

import (
	"context"
	"net/url"
	"time"

	pb "github.com/dusk-network/dusk-protobuf/autogen/go/monitor"
	"github.com/dusk-network/dusk-wallet/v2/block"
	g "google.golang.org/grpc"
)

type Client struct {
	tgt       *url.URL
	lastBlock *block.Block
}

// New creates a Client
// TODO: add TLS certificates
// TODO: subscribe to all relevant topics
func New(uri *url.URL) *Client {
	return &Client{uri, nil}
}

func (c *Client) sendOnce(callback func(pb.MonitorClient, context.Context) error) error {
	// URL.Host returns hostname:port
	conn, err := g.Dial(c.tgt.Host, g.WithInsecure(), g.WithBlock())
	if err != nil {
		return err
	}

	defer conn.Close()
	mon := pb.NewMonitorClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)

	defer cancel()
	return callback(mon, ctx)
}
