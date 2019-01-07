package connmgr_test

import (
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/connmgr"
)

func TestConnect(t *testing.T) {
	cfg := connmgr.Config{
		GetAddress:   nil,
		OnConnection: nil,
		OnAccept:     nil,
		Port:         "",
		DialTimeout:  0,
	}

	cm := connmgr.New(cfg)
	cm.Run()

	ipport := "google.com:80"

	r := connmgr.Request{Addr: ipport}

	err := cm.Connect(&r)
	assert.Equal(t, nil, err)
	assert.Equal(t, 1, len(cm.ConnectedList))

}
func TestNewRequest(t *testing.T) {

	address := payload.NewNetAddress("216.58.212.174", 80)

	var getAddr = func() (*payload.NetAddress, error) {
		return address, nil
	}

	cfg := connmgr.Config{
		GetAddress:   getAddr,
		OnConnection: nil,
		OnAccept:     nil,
		Port:         "",
		DialTimeout:  0,
	}

	cm := connmgr.New(cfg)

	cm.Run()

	cm.NewRequest()

	if _, ok := cm.ConnectedList[address.String()]; ok {
		assert.Equal(t, true, ok)
		assert.Equal(t, 1, len(cm.ConnectedList))
		return
	}

	assert.Fail(t, "Could not find the address in the connected lists")

}
func TestDisconnect(t *testing.T) {

	address := payload.NewNetAddress("216.58.212.174", 80)

	var getAddr = func() (*payload.NetAddress, error) {
		return address, nil
	}

	cfg := connmgr.Config{
		GetAddress:   getAddr,
		OnConnection: nil,
		OnAccept:     nil,
		Port:         "",
		DialTimeout:  0,
	}

	cm := connmgr.New(cfg)

	cm.Run()
	cm.NewRequest()
	cm.Disconnect(address.String())

	assert.Equal(t, 0, len(cm.ConnectedList))
}
