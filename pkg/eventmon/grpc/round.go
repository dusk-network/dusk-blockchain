package grpc

import (
	"context"
	"time"

	pb "github.com/dusk-network/dusk-protobuf/autogen/go/monitor"
	"github.com/dusk-network/dusk-wallet/v2/block"
)

//NotifyBlockUpdate opens a connection to the monitoring server any time there
//is an update. It is questionable whether this should be a stream instead
func (c *Client) NotifyBlockUpdate(blk *block.Block) error {
	return c.sendOnce(func(mon pb.MonitorClient, ctx context.Context) error {
		now := time.Now()

		blockUpdate := &pb.BlockUpdate{
			Height:    blk.Header.Height,
			Hash:      blk.Header.Hash,
			Timestamp: now.Format(time.RFC3339),
			TxAmount:  uint32(len(blk.Txs)),
		}

		if c.lastBlock != nil && (blk.Header.Height-c.lastBlock.Header.Height) == 1 {
			diff := getDiffInSeconds(blk.Header.Timestamp, c.lastBlock.Header.Timestamp)
			blockUpdate.BlockTimeSec = diff
		}

		c.lastBlock = blk
		_, err := mon.NotifyBlock(ctx, blockUpdate)
		return err
	})
}

func getDiffInSeconds(currentTimeStamp int64, lastTimeStamp int64) uint32 {
	lastTime := time.Unix(lastTimeStamp, 0)
	currentTime := time.Unix(currentTimeStamp, 0)
	return uint32(currentTime.Sub(lastTime).Seconds())
}
