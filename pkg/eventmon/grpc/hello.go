package grpc

import (
	"context"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	pb "github.com/dusk-network/dusk-protobuf/autogen/go/monitor"
)

// Hello notifies the monitor about this node's version
func (c *Client) Hello() error {
	return c.sendOnce(func(mon pb.MonitorClient, ctx context.Context) error {
		ver := &pb.SemverRequest{
			Major: uint32(protocol.NodeVer.Major),
			Minor: uint32(protocol.NodeVer.Minor),
			Patch: uint32(protocol.NodeVer.Patch),
		}

		_, err := mon.Hello(ctx, ver)
		return err
	})
}

func (c *Client) Bye() error {
	return c.sendOnce(func(mon pb.MonitorClient, ctx context.Context) error {
		_, err := mon.Bye(ctx, &pb.EmptyRequest{})
		return err
	})
}
