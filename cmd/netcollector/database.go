// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package main

import (
	"sort"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type stats struct {
	min, max uint64
	addrs    map[string]bool
}

type database struct {

	// notes map schema:
	// key = network_msgType_msgId (e.g kadcast_tx_49229cce9260d9c28, gossip_tx_49229cce9260d9c28 )
	// value =  min, max and slice of addresses

	messages map[string]*stats
	lock     sync.RWMutex
}

func (d *database) HandleNote(args *NoteArgs) error {

	d.lock.Lock()
	defer d.lock.Unlock()

	if d.messages == nil {
		d.messages = make(map[string]*stats)
	}

	if _, ok := d.messages[args.MsgID]; !ok {
		d.messages[args.MsgID] = &stats{
			max:   args.Recv_at,
			min:   args.Recv_at,
			addrs: make(map[string]bool)}

		d.messages[args.MsgID].addrs[args.Addr] = true
		return nil
	}

	if _, ok := d.messages[args.MsgID].addrs[args.Addr]; ok {
		// This node address has already registered a note with this messageID
		return nil
	}

	d.messages[args.MsgID].addrs[args.Addr] = true

	if d.messages[args.MsgID].max < args.Recv_at {
		d.messages[args.MsgID].max = args.Recv_at
	}

	if d.messages[args.MsgID].min > args.Recv_at {
		d.messages[args.MsgID].min = args.Recv_at
	}

	return nil
}

// printTPS calculates and prints Transactions Per Second metric
// TPS = nodesCount/(lastTimeStamp - firstTimeStamp)
func (d *database) printTPS(interval time.Duration) {
	for {
		time.Sleep(interval)

		if len(d.messages) == 0 {
			continue
		}

		log.Infof("Stats Snapshot %d --------------------", time.Now().Unix())

		// Sort by RecvTime
		keys := make([]string, 0, len(d.messages))
		for k := range d.messages {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		d.lock.RLock()
		var averageTPS float64
		for _, key := range keys {
			m := d.messages[key]
			nodesCount := len(m.addrs)

			// Time in milliseconds between first and last registration of the message
			delayMilliSec := float64((m.max - m.min)) / float64(time.Millisecond)
			tps := float64(nodesCount) / (delayMilliSec / 1000)

			averageTPS += tps

			// Number of $nodesCount nodes received the wire message within $Time milliseconds

			log.WithField("msg", key).
				WithField("nodesCount", nodesCount).
				WithField("time_ms", delayMilliSec).Infof("TPS %.2f", tps)
		}

		log.Infof("average TPS %f", averageTPS/float64(len(d.messages)))

		d.lock.RUnlock()
	}
}
