package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	pb "github.com/dusk-network/dusk-protobuf/autogen/go/monitor"
	g "google.golang.org/grpc"
)

type Client struct {
	tgt string
}

func New(host, port string) *Client {
	return &Client{fmt.Sprintf("%s:%s", host, port)}
}

func (c *Client) Hello() error {
	conn, err := g.Dial(c.tgt, g.WithInsecure(), g.WithBlock())
	if err != nil {
		return err
	}

	defer conn.Close()
	mon := pb.NewMonitorClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)

	defer cancel()

	ver := &pb.SemverRequest{
		Major: uint32(protocol.NodeVer.Major),
		Minor: uint32(protocol.NodeVer.Minor),
		Patch: uint32(protocol.NodeVer.Patch),
	}

	if _, err := mon.Hello(ctx, ver); err != nil {
		return err
	}

	return nil
}
