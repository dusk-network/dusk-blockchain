package grpc

import (
	"context"
	"time"

	pb "github.com/dusk-network/dusk-protobuf/autogen/go/monitor"
	"github.com/dusk-network/dusk-wallet/v2/block"
)

//NotifyBlockUpdate sends an alert with the Block data any time a new block is
//finalized and added to the blockchain. It opens a connection to the monitoring
// server any time there is an update. It is questionable whether this should
// be a stream instead
func (c *Client) NotifyBlockUpdate(parent context.Context, blk block.Block) error {
	return c.send(parent, func(mon pb.MonitorClient, ctx context.Context) error {
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

// NotifyBlockSlowdown sends an alert with the data about the slowdown in
// accepting blocks
func (c *Client) NotifyBlockSlowdown(parent context.Context) error {
	return c.send(parent, func(mon pb.MonitorClient, ctx context.Context) error {
		if c.lastBlock != nil {
			t := time.Since(time.Unix(c.lastBlock.Header.Timestamp, int64(0)))
			alert := &pb.SlowdownAlert{
				LastKnownHash:         c.lastBlock.Header.Hash,
				LastKnownHeight:       c.lastBlock.Header.Height,
				TimeSinceLastBlockSec: uint32(t.Seconds()),
			}
			_, err := mon.NotifySlowdown(ctx, alert)
			return err
		}
		_, err := mon.NotifySlowdown(ctx, &pb.SlowdownAlert{})
		return err
	})
}
