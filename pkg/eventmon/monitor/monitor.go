package monitor

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/bwesterb/go-ristretto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/agreement"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/events"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/reduction"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/selection"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/eventmon"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

func LaunchMonitor(eventbus wire.EventBroker, srvUrl string, bid ristretto.Scalar) *broker {
	broker := newBroker(eventbus, srvUrl)
	go broker.monitor(bid)
	return broker
}

type (
	broker struct {
		roundChan     <-chan uint64
		bestScoreChan <-chan *selection.ScoreEvent
		reductionChan <-chan *events.Reduction
		agreementChan <-chan *events.AggregatedAgreement
		logChan       <-chan *eventmon.Event

		msgChan  chan<- string
		quitChan chan<- struct{}
		conn     net.Conn

		blockInfo *info
	}

	info struct {
		round     uint64
		timestamp time.Time
		blockTime time.Duration
		hash      []byte
		score     uint64
	}
)

func newBroker(eventbus wire.EventBroker, url string) *broker {
	msgChan, quitChan, conn := newClient(url)
	roundChan := consensus.InitRoundUpdate(eventbus)

	return &broker{
		roundChan:     roundChan,
		bestScoreChan: selection.LaunchNotification(eventbus),
		reductionChan: reduction.LaunchNotification(eventbus),
		agreementChan: agreement.LaunchNotification(eventbus),
		msgChan:       msgChan,
		quitChan:      quitChan,
		conn:          conn,
	}
}

func (b *broker) monitor(bb ristretto.Scalar) {
	// b.msgChan <- "bid:" + bb.String()
	firstRound := <-b.roundChan
	b.blockInfo = newInfo(firstRound)
	for {
		select {
		case round := <-b.roundChan:
			b.blockInfo.blockTime = time.Since(b.blockInfo.timestamp)
			b.msgChan <- blockMsg(b.blockInfo)
			b.blockInfo = newInfo(round)
		case bestScore := <-b.bestScoreChan:
			b.blockInfo.score = big.NewInt(0).SetBytes(bestScore.Score).Uint64()
			b.blockInfo.hash = bestScore.VoteHash
			b.msgChan <- "status:selection"
		case redEvent := <-b.reductionChan:
			b.blockInfo.hash = redEvent.BlockHash
			b.msgChan <- "status:reduction"
		case agreement := <-b.agreementChan:
			// TODO if different hash send
			b.blockInfo.hash = agreement.BlockHash
			b.msgChan <- "status:agreement"
		case logEvent := <-b.logChan:
			if logEvent.Severity == eventmon.Warn || logEvent.Severity == eventmon.Err {
				b.msgChan <- logEvent.Msg
			}
		}
	}
}

func newInfo(round uint64) *info {
	return &info{
		round:     round,
		timestamp: time.Now(),
	}
}

func blockMsg(i *info) string {
	var str strings.Builder
	str.WriteString("type:block,round:")
	str.WriteString(strconv.FormatUint(i.round, 10))
	str.WriteString(",hash:")
	str.WriteString(hex.EncodeToString(i.hash))
	str.WriteString(",block_time:")
	str.WriteString(fmt.Sprintf("%f", i.blockTime.Seconds()))
	str.WriteString(",timestamp:")
	str.WriteString(i.timestamp.Format(time.RFC3339))
	str.WriteString(",score:")
	str.WriteString(strconv.FormatUint(i.score, 10))
	return str.String()
}

var _empty struct{}

func newClient(url string) (chan<- string, chan<- struct{}, net.Conn) {
	payloadChan := make(chan string, 1)
	quitChan := make(chan struct{}, 1)

	conn, err := net.Dial("tcp", url)
	if err != nil {
		panic(err)
	}
	go func() {
		for {
			select {
			case payload := <-payloadChan:
				fmt.Fprintf(conn, payload+"\n")
			case <-quitChan:
				_ = conn.Close()
				return
			}
		}
	}()
	return payloadChan, quitChan, conn
}
