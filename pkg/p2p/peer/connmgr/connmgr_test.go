package connmgr_test

import (
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/connmgr"
	"io/ioutil"
	"net"
	"testing"
)

func init() {
	log.SetOutput(ioutil.Discard)
}

func TestConnect(t *testing.T) {
	cfg := connmgr.Config{
		GetAddress:   nil,
		OnConnection: nil,
		OnAccept:     nil,
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
	address := "google.com:80"

	var getAddr = func() (string, error) {
		return address, nil
	}

	cfg := connmgr.Config{
		GetAddress: getAddr,
	}

	cm := connmgr.New(cfg)

	cm.Run()
	cm.NewRequest()

	if _, ok := cm.ConnectedList[address]; ok {
		assert.Equal(t, true, ok)
		assert.Equal(t, 1, len(cm.ConnectedList))
		return
	}

	assert.Fail(t, "Could not find the address in the connected lists")
}

func TestDisconnect(t *testing.T) {
	address := "google.com:80"

	var getAddr = func() (string, error) {
		return address, nil
	}

	cfg := connmgr.Config{
		GetAddress:   getAddr,
		OnConnection: nil,
		OnAccept:     nil,
	}

	cm := connmgr.New(cfg)

	_, conn := net.Pipe()
	cm.ConnectedList[address] = &connmgr.Request{Conn: conn, Addr: address}
	cm.Run()
	cm.NewRequest()
	cm.Disconnect(address)

	assert.Equal(t, 0, len(cm.ConnectedList))
}
