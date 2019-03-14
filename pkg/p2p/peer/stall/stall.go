package stall

import (
	"sync"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/commands"
)

// Detector is stall detector that will keep track of all pending messages.
// If any message takes too long to reply the detector will disconnect the peer.
type Detector struct {
	responseTime time.Duration
	tickInterval time.Duration

	lock      sync.Mutex
	responses map[topics.Topic]time.Time

	// The detector is embedded into a peer and the peer watches this quit chan
	// If this chan is closed, the peer disconnects
	Quitch chan struct{}

	// atomic vals
	disconnected int32
}

// NewDetector returns a new Detector.
// rT is the responseTime and signals how long a peer has to reply back to a sent message.
// tickerInterval is how often the detector wil check for stalled messages.
func NewDetector(rTime time.Duration, tickerInterval time.Duration) *Detector {
	d := &Detector{
		responseTime: rTime,
		tickInterval: tickerInterval,
		lock:         sync.Mutex{},
		responses:    map[topics.Topic]time.Time{},
		Quitch:       make(chan struct{}),
	}
	go d.loop()
	return d
}

func (d *Detector) loop() {
	ticker := time.NewTicker(d.tickInterval)

loop:
	for {
		select {
		case <-ticker.C:
			now := time.Now()
			d.lock.Lock()
			for _, deadline := range d.responses {
				if now.After(deadline) {
					log.WithField("prefix", "stall").Info("Deadline passed")
					ticker.Stop()
					d.lock.Unlock()
					break loop
				}
			}
			d.lock.Unlock()
		}
	}
	d.Quit()
	d.DeleteAll()
	ticker.Stop()
}

// Quit is a concurrent safe way to call the Quit channel without blocking
func (d *Detector) Quit() {
	// return if already disconnected
	if atomic.LoadInt32(&d.disconnected) != 0 {
		return
	}
	atomic.AddInt32(&d.disconnected, 1)
	close(d.Quitch)
}

// AddMessage adds a msg to the Detector.
// Call this function when we send a message to a peer.
// The command passed through is the command that we sent and not the command we expect to receive.
func (d *Detector) AddMessage(cmd topics.Topic) {
	cmds := d.addMessage(cmd)
	d.lock.Lock()
	for _, cmd := range cmds {
		d.responses[cmd] = time.Now().Add(d.responseTime)
	}
	d.lock.Unlock()
}

// RemoveMessage removes a msg from the Detector.
// Call this function when we receive a message from a peer.
// This will remove the pendingresponse message from the map.
// The command passed through is the command we received.
func (d *Detector) RemoveMessage(cmd topics.Topic) {
	cmds := d.removeMessage(cmd)
	d.lock.Lock()
	for _, cmd := range cmds {
		delete(d.responses, cmd)
	}
	d.lock.Unlock()
}

// DeleteAll empties the map of all contents and
// is called when the detector is being shut down
func (d *Detector) DeleteAll() {
	d.lock.Lock()
	d.responses = make(map[topics.Topic]time.Time)
	d.lock.Unlock()
}

// GetMessages will return a map of all of the pendingResponses and their deadlines
func (d *Detector) GetMessages() map[topics.Topic]time.Time {
	var resp map[topics.Topic]time.Time
	d.lock.Lock()
	resp = d.responses
	d.lock.Unlock()
	return resp
}

// When a message is added, we will add a deadline for expected response
func (d *Detector) addMessage(cmd topics.Topic) []topics.Topic {

	cmds := []topics.Topic{}

	switch cmd {
	case topics.Headers, topics.GetHeaders:
		// We now will expect a Headers Message
		cmds = append(cmds, topics.Headers)
	case topics.Addr, topics.GetAddr:
		// We now will expect a Addr Message
		cmds = append(cmds, topics.Addr)
	case topics.GetData:
		// We will now expect a block/tx message
		// We can optimise this by including the exact inventory type, however it is not needed
		cmds = append(cmds, topics.Block)
		cmds = append(cmds, topics.Tx)
	case topics.Inv, topics.GetBlocks:
		// we will now expect a inv message
		cmds = append(cmds, topics.Inv)
	case topics.VerAck, topics.Version:
		// We will now expect a verack
		cmds = append(cmds, topics.VerAck)
	}
	return cmds
}

// If receive a message, we will delete it from pending
func (d *Detector) removeMessage(cmd topics.Topic) []topics.Topic {

	cmds := []topics.Topic{}

	switch cmd {
	case topics.Block:
		// We will now expect a block/tx message
		cmds = append(cmds, topics.Block)
		cmds = append(cmds, topics.Tx)
	case topics.Tx:
		// We will now expect a block/tx message
		cmds = append(cmds, topics.Block)
		cmds = append(cmds, topics.Tx)
	case topics.GetBlocks:
		// we will now expect a inv message
		cmds = append(cmds, topics.Inv)
	default:
		// We will now expect a verack
		cmds = append(cmds, cmd)
	}
	return cmds
}
