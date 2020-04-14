package grpc

import (
	"context"
	"net/url"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	pb "github.com/dusk-network/dusk-protobuf/autogen/go/monitor"
	g "google.golang.org/grpc"
)

// Client of the grpc remote monitoring server
type Client struct {
	tgt       *url.URL
	lastBlock *block.Block
}

type sendCB func(pb.MonitorClient, context.Context) error

// New creates a Client
// TODO: add TLS certificates
func New(uri *url.URL) *Client {
	return &Client{uri, nil}
}

func (c *Client) send(parentCtx context.Context, callback sendCB) error {
	// URL.Host returns hostname:port
	conn, err := g.Dial(c.tgt.Host, g.WithInsecure(), g.WithBlock())
	if err != nil {
		return err
	}

	defer func() {
		_ = conn.Close()
	}()
	mon := pb.NewMonitorClient(conn)
	ctx, cancel := context.WithTimeout(parentCtx, 2*time.Second)

	defer cancel()
	return callback(mon, ctx)
}

func (c *Client) sendUnlinked(callback func(pb.MonitorClient, context.Context) error) error {
	return c.send(context.Background(), callback)
}
