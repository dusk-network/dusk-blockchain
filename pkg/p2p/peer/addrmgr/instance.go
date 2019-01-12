package addrmgr

import (
	cnf "github.com/spf13/viper"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"strconv"
	"strings"
	"sync"
)

var instance *Addrmgr
var once sync.Once

// GetInstance creates a Addrmgr instance as a Singleton
func GetInstance() *Addrmgr {

	if instance == nil {
		once.Do(func() {
			instance = New()
			instance.AddAddrs(getPermanentAddresses())
		})
	}

	return instance
}

func getPermanentAddresses() []*payload.NetAddress {
	// Add the (permanent) seed addresses to the Address Manager
	var netAddrs []*payload.NetAddress
	addrs := cnf.GetStringSlice("net.peer.seeds")
	for _, addr := range addrs {
		s := strings.Split(addr, ":")
		port, _ := strconv.ParseUint(s[1], 10, 16)
		netAddrs = append(netAddrs, payload.NewNetAddress(s[0], uint16(port)))
	}
	return netAddrs
}
