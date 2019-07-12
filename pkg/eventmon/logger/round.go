package logger

import (
	"encoding/hex"
	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
)

func (l *LogProcessor) PublishBlockEvent(blk *block.Block) {
	e := l.WithBlock(blk)
	e.Infoln("New Block Accepted")
}

func (l *LogProcessor) withRoundCode(fields log.Fields) *log.Entry {
	return l.WithTime(fields).WithField("code", "round")
}

func (l *LogProcessor) WithBlock(blk *block.Block) *log.Entry {
	fields := log.Fields{
		"round":     blk.Header.Height,
		"blockHash": hex.EncodeToString(blk.Header.Hash),
		"numtxs":    len(blk.Txs),
	}
	entry := l.withRoundCode(fields)

	if l.lastBlock != nil && (blk.Header.Height-l.lastBlock.Header.Height) == 1 {
		blockTimeMs := getDiffInMs(blk.Header.Timestamp, l.lastBlock.Header.Timestamp)
		entry = entry.WithField("blockTime", blockTimeMs)
	}

	l.lastBlock = blk

	return entry
}

func getDiffInMs(currentTimeStamp int64, lastTimeStamp int64) int {
	lastTime := time.Unix(lastTimeStamp, 0)
	currentTime := time.Unix(currentTimeStamp, 0)
	diffSeconds := currentTime.Sub(lastTime).Seconds()

	// Return float64 seconds value as integer millisecond value
	return int(diffSeconds * 1000)
}
