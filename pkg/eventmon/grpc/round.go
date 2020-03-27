package grpc

import (
	"context"
	"time"

	pb "github.com/dusk-network/dusk-protobuf/autogen/go/monitor"
	"github.com/dusk-network/dusk-wallet/v2/block"
)

//NotifyBlockUpdate opens a connection to the monitoring server any time there
//is an update. It is questionable whether this should be a stream instead
func (c *Client) NotifyBlockUpdate(blk block.Block) error {
	return c.sendOnce(func(mon pb.MonitorClient, ctx context.Context) error {
		blockUpdate := &pb.BlockUpdate{
			Height:    blk.Header.Height,
			Hash:      blk.Header.Hash,
			Timestamp: blk.Header.Timestamp,
			TxAmount:  uint32(len(blk.Txs)),
		}

		if c.lastBlock != nil && (blk.Header.Height-c.lastBlock.Header.Height) == 1 {
			blockUpdate.BlockTimeSec = uint32(blk.Header.Timestamp - c.lastBlock.Header.Timestamp)
		}

		c.lastBlock = &blk
		_, err := mon.NotifyBlock(ctx, blockUpdate)
		return err
	})
}

func (c *Client) NotifyBlockSlowdown() error {
	return c.sendOnce(func(mon pb.MonitorClient, ctx context.Context) error {
		if c.lastBlock != nil {
			t := time.Now().Sub(time.Unix(c.lastBlock.Header.Timestamp, int64(0)))
			alert := &pb.SlowdownAlert{
				// TODO: TimeSinceLastBlockSec
				// TODO: LastKnownHash
				// TODO: LastKnownHeight
				TimeSinceLastBlock: uint32(t.Seconds()),
			}
			_, err := mon.NotifySlowdown(ctx, alert)
			return err
		}

		_, err := mon.NotifySlowdown(ctx, &pb.SlowdownAlert{TimeSinceLastBlock: uint32(0)})
		return err
	})
}
