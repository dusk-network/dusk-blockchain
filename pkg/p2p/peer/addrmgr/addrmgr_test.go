package addrmgr_test

import (
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/addrmgr"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
)

var privateRand *rand.Rand

func init() {
	privateRand = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func TestNewAddrs(t *testing.T) {

	addrmgr := addrmgr.New()
	assert.NotEqual(t, nil, addrmgr)
}
func TestAddAddrs(t *testing.T) {

	addrmgr := addrmgr.New()

	ip := "0000:0000:0000:0000:0000:0000:0000:0000"
	netAddr1 := payload.NewNetAddress(ip, 1033)
	msgAddr1 := payload.NewMsgAddr()
	msgAddr1.AddAddr(netAddr1)

	ip = "0000:0000:0000:0000:0000:0000:0000:0000"
	netAddr2 := payload.NewNetAddress(ip, 1033) // same
	msgAddr2 := payload.NewMsgAddr()
	msgAddr2.AddAddr(netAddr2)

	ip = "1000:0000:0000:0000:0000:0000:0000:0000"
	netAddr3 := payload.NewNetAddress(ip, 1033) // different
	msgAddr3 := payload.NewMsgAddr()
	msgAddr3.AddAddr(netAddr3)

	addrs := []*payload.NetAddress{netAddr1, netAddr2, netAddr3}
	addrmgr.AddAddrs(addrs)

	assert.Equal(t, 2, len(addrmgr.Unconnected()))
	assert.Equal(t, 0, len(addrmgr.Good()))
	assert.Equal(t, 0, len(addrmgr.Bad()))
}

func TestFetchMoreAddress(t *testing.T) {
	var ip string
	addrmgr := addrmgr.New()

	addrs := []*payload.NetAddress{}

	for i := 0; i <= 2000; i++ { // Add more than maxAllowedAddrs
		ip = ipV6Address()

		addr := payload.NewNetAddress(ip, 1033)
		addrs = append(addrs, addr)
	}

	addrmgr.AddAddrs(addrs)

	assert.Equal(t, false, addrmgr.FetchMoreAddresses())

}
func TestConnComplete(t *testing.T) {

	addrmgr := addrmgr.New()

	ip := "1000:0000:0000:0000:0000:0000:0000:0000" // different
	addr := payload.NewNetAddress(ip, 1033)

	//ip2 := [16]byte{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0} // different
	ip2 := "2000:0000:0000:0000:0000:0000:0000:0000"
	addr2 := payload.NewNetAddress(ip2, 1033)

	ip3 := "3000:0000:0000:0000:0000:0000:0000:0000" // different
	addr3 := payload.NewNetAddress(ip3, 1033)

	addrs := []*payload.NetAddress{addr, addr2, addr3}

	addrmgr.AddAddrs(addrs)

	assert.Equal(t, len(addrs), len(addrmgr.Unconnected()))

	// a successful connection
	addrmgr.ConnectionComplete(addr.String(), true)
	addrmgr.ConnectionComplete(addr.String(), true) // should have no change

	assert.Equal(t, len(addrs)-1, len(addrmgr.Unconnected()))
	assert.Equal(t, 1, len(addrmgr.Good()))

	// another successful connection
	addrmgr.ConnectionComplete(addr2.String(), true)

	assert.Equal(t, len(addrs)-2, len(addrmgr.Unconnected()))
	assert.Equal(t, 2, len(addrmgr.Good()))

}
func TestAttempted(t *testing.T) {

	addrmgr := addrmgr.New()

	ip := "1000:0000:0000:0000:0000:0000:0000:0000" // different
	addr := payload.NewNetAddress(ip, 1033)

	addrs := []*payload.NetAddress{addr}

	addrmgr.AddAddrs(addrs)

	addrmgr.Failed(addr.String())

	assert.Equal(t, 1, len(addrmgr.Bad())) // newAddrs was attmepted and failed. Move to Bad

}
func TestAttemptedMoveFromGoodToBad(t *testing.T) {

	addrmgr := addrmgr.New()

	ip := "1000:0000:0000:0000:0000:0000:0000:0000" // different
	addr := payload.NewNetAddress(ip, 1043)

	addrs := []*payload.NetAddress{addr}

	addrmgr.AddAddrs(addrs)

	addrmgr.ConnectionComplete(addr.String(), true)
	addrmgr.ConnectionComplete(addr.String(), true)
	addrmgr.ConnectionComplete(addr.String(), true)

	assert.Equal(t, 1, len(addrmgr.Good()))

	addrmgr.Failed(addr.String())
	addrmgr.Failed(addr.String())
	addrmgr.Failed(addr.String())
	addrmgr.Failed(addr.String())
	addrmgr.Failed(addr.String())
	addrmgr.Failed(addr.String())
	addrmgr.Failed(addr.String())
	addrmgr.Failed(addr.String())
	addrmgr.Failed(addr.String())
	// over threshhold, and will be classed as a badAddr L251

	assert.Equal(t, 0, len(addrmgr.Good()))
	assert.Equal(t, 1, len(addrmgr.Bad()))
}

func TestGetAddress(t *testing.T) {

	addrmgr := addrmgr.New()

	ip := "1000:0000:0000:0000:0000:0000:0000:0000" // different
	addr := payload.NewNetAddress(ip, 10333)
	ip2 := "1000:0000:0000:0000:0000:0000:0000:0000" // different
	addr2 := payload.NewNetAddress(ip2, 10334)
	ip3 := "1000:0000:0000:0000:0000:0000:0000:0000" // different
	addr3 := payload.NewNetAddress(ip3, 10335)

	addrs := []*payload.NetAddress{addr, addr2, addr3}

	addrmgr.AddAddrs(addrs)

	fetchAddr, err := addrmgr.NewAddr()
	assert.Equal(t, nil, err)

	ipports := []string{addr.String(), addr2.String(), addr3.String()}

	assert.Contains(t, ipports, fetchAddr)
}

// ipV6Address returns a valid IPv6 address as net.IP
// TODO: If used in more tests make it available
func ipV6Address() string {
	var ip net.IP
	for i := 0; i < net.IPv6len; i++ {
		number := uint8(privateRand.Intn(255))
		ip = append(ip, number)
	}
	return ip.String()
}
