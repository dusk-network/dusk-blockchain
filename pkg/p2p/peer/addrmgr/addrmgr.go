package addrmgr

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/noded/config"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

const (
	maxTries = 5 // Try to connect five times, if we cannot then we say it is bad
	// We will keep bad addresses, so that we do not attempt to connect to them in the future.
	// nodes at the moment seem to send a large percentage of bad nodes.

	maxFailures = 12 // This will be incremented when we connect to a node and for whatever reason, they keep disconnecting.
	// The number is high because there could be a period of time, where a node is behaving unrepsonsively and
	// we do not want to immmediately put it inside of the bad bucket.

	// This is the maximum amount of addresses that the addrmgr will hold
	maxAllowedAddrs = 2000
)

type addrStats struct {
	tries       uint8
	failures    uint8
	permanent   bool // permanent = inbound, temp = outbound
	lastTried   time.Time
	lastSuccess time.Time
}

// Addrmgr holds maps to keep info of new, good, bad addresses
type Addrmgr struct {
	addrmtx   sync.RWMutex
	goodAddrs map[*wire.NetAddress]addrStats
	newAddrs  map[*wire.NetAddress]struct{}
	badAddrs  map[*wire.NetAddress]addrStats
	knownList map[string]*wire.NetAddress // this contains all of the addresses in badAddrs, newAddrs and goodAddrs
}

// New creates a new address manager
func New() *Addrmgr {
	am := &Addrmgr{
		sync.RWMutex{},
		make(map[*wire.NetAddress]addrStats, 100),
		make(map[*wire.NetAddress]struct{}, 100),
		make(map[*wire.NetAddress]addrStats, 100),
		make(map[string]*wire.NetAddress, 100),
	}
	return am
}

func (a *Addrmgr) getPermanentAddresses() []*wire.NetAddress {
	// Add the (permanent) seed addresses to the Address Manager
	var netAddrs []*wire.NetAddress
	addrs := config.EnvNetCfg.Peer.Seeds
	for _, addr := range addrs {
		s := strings.Split(addr, ":")
		port, _ := strconv.ParseUint(s[1], 10, 16)
		netAddrs = append(netAddrs, wire.NewNetAddress(s[0], uint16(port)))
	}
	return netAddrs
}

// AddAddrs will add new addresses into the newaddr list
// This is safe for concurrent access.
func (a *Addrmgr) AddAddrs(newAddrs []*wire.NetAddress) {

	newAddrs = removeDuplicates(newAddrs)

	var nas []*wire.NetAddress
	for _, addr := range newAddrs {
		a.addrmtx.Lock()

		if _, ok := a.knownList[addr.String()]; !ok { // filter unique addresses
			nas = append(nas, addr)
		}
		a.addrmtx.Unlock()
	}

	for _, addr := range nas {
		a.addrmtx.Lock()
		a.newAddrs[addr] = struct{}{}
		a.knownList[addr.String()] = addr
		a.addrmtx.Unlock()
	}
}

// Good returns the good addresses.
// A good address is:
// - Known
// - has been successfully connected to in the last week
// - Note: that while users are launching full nodes from their laptops, the one week marker will need to be modified.
func (a *Addrmgr) Good() []wire.NetAddress {

	var goodAddrs []wire.NetAddress

	var oneWeekAgo = time.Now().Add(((time.Hour * 24) * 7) * -1)

	a.addrmtx.RLock()
	// TODO: sort addresses, permanent ones go first
	for addr, stat := range a.goodAddrs {
		if stat.lastTried.Before(oneWeekAgo) {
			continue
		}
		goodAddrs = append(goodAddrs, *addr)

	}
	a.addrmtx.RUnlock()

	return goodAddrs
}

// Unconnected addresses are addresses in the newAddr list.
func (a *Addrmgr) Unconnected() []wire.NetAddress {

	var unconnAddrs []wire.NetAddress

	a.addrmtx.RLock()
	for addr := range a.newAddrs {
		unconnAddrs = append(unconnAddrs, *addr)
	}
	a.addrmtx.RUnlock()
	return unconnAddrs
}

// Bad addresses are addresses in the badAddr list.
// They are put there if we have tried to connect
// to them in the past and it failed.
func (a *Addrmgr) Bad() []wire.NetAddress {

	var badAddrs []wire.NetAddress

	a.addrmtx.RLock()
	for addr := range a.badAddrs {
		badAddrs = append(badAddrs, *addr)
	}
	a.addrmtx.RUnlock()
	return badAddrs
}

// FetchMoreAddresses will return true if the numOfKnownAddrs are less than 100
// This number is kept low because at the moment, there are not a lot of Good Addresses
// Tests have shown that at most there are < 100
func (a *Addrmgr) FetchMoreAddresses() bool {

	var numOfKnownAddrs int

	a.addrmtx.RLock()
	numOfKnownAddrs = len(a.knownList)
	a.addrmtx.RUnlock()

	return numOfKnownAddrs < maxAllowedAddrs
}

// ConnectionComplete will be called by the server when we have done the handshake
// and NOT when we have successfully connected with net.Conn.
// It is to tell the AddrMgr that we have connected to a peer.
// This peer should already be known to the AddrMgr because we send it to the Connmgr
func (a *Addrmgr) ConnectionComplete(addr string, inbound bool) {
	a.addrmtx.Lock()
	defer a.addrmtx.Unlock()

	// If addrmgr does not know this address, then we just return
	if _, ok := a.knownList[addr]; !ok {
		log.Infof("Connected to an unknown addr:port %s", addr)
		return
	}

	na := a.knownList[addr]

	// Move it from newAddrs List to goodAddrs List
	stats := a.goodAddrs[na]
	stats.lastSuccess = time.Now()
	stats.lastTried = time.Now()
	stats.permanent = inbound
	stats.tries++
	a.goodAddrs[na] = stats

	// remove it from new and bad, if it was there
	delete(a.newAddrs, na)
	delete(a.badAddrs, na)

	// Unfortunately, Will have a lot of bad nodes because of people connecting to nodes
	// from their laptop. TODO Sort function in good will mitigate.

}

// Failed will be called by ConnMgr
// It is used to tell the AddrMgr that they had tried connecting an address and have failed
// This is concurrent safe
func (a *Addrmgr) Failed(addr string) {
	a.addrmtx.Lock()
	defer a.addrmtx.Unlock()

	// if addrmgr does not know this, then we just return
	if _, ok := a.knownList[addr]; !ok {
		log.Info("Connected to an unknown address:port ", addr)
		return
	}

	na := a.knownList[addr]

	// HMM: logic here could be simpler if we make it one list instead

	if stats, ok := a.badAddrs[na]; ok {
		stats.lastTried = time.Now()
		stats.failures++
		stats.tries++
		if float32(stats.failures)/float32(stats.tries) > 0.8 && stats.tries > 5 {
			delete(a.badAddrs, na)
			return
		}
		a.badAddrs[na] = stats
		return
	}

	if stats, ok := a.goodAddrs[na]; ok {
		log.Debug("We have a good addr", na.String())
		stats.lastTried = time.Now()
		stats.failures++
		stats.tries++
		if float32(stats.failures)/float32(stats.tries) > 0.5 && stats.tries > 10 {
			delete(a.goodAddrs, na)
			a.badAddrs[na] = stats
			return
		}
		a.goodAddrs[na] = stats
		return
	}

	if _, ok := a.newAddrs[na]; ok {
		delete(a.newAddrs, na)
		a.badAddrs[na] = addrStats{}
	}

}

// GetGoodAddresses give the best addresses we have from good.
// TODO: Remove, is duplicate of Good()
func (a *Addrmgr) GetGoodAddresses() []wire.NetAddress {
	a.addrmtx.RLock()
	defer a.addrmtx.RUnlock()
	return a.Good()
}

// NewAddres will return an address for the caller to connect to.
// In our case, it will be the connection manager.
func (a *Addrmgr) NewAddres() (string, error) {
	// For now it just returns a random value from unconnected
	// TODO: When an address is tried, the address manager is notified.
	// When asked for a new address, this should be taken into account
	// when choosing a new one, also the number of retries.
	unconnected := a.Unconnected()
	if len(unconnected) == 0 {
		return "", fmt.Errorf("Failed to issue a new peer address to connect to")
	}
	rand := rand.Intn(len(unconnected))
	return unconnected[rand].String(), nil
}

// https://www.dotnetperls.com/duplicates-go
func removeDuplicates(elements []*wire.NetAddress) []*wire.NetAddress {

	encountered := map[string]bool{}
	result := []*wire.NetAddress{}

	for _, element := range elements {
		if encountered[element.String()] == true {
			// Do not add duplicate.
		} else {
			// Record this element as an encountered element.
			encountered[element.String()] = true
			// Append to result slice.
			result = append(result, element)
		}
	}
	// Return the new slice.
	return result
}
