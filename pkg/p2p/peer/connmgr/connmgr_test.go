package connmgr_test

import (
	"bou.ke/monkey"
	log "github.com/sirupsen/logrus"
	cfg "github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/peer/connmgr"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"io/ioutil"
	"net"
	"reflect"
	"testing"
)

const tmpUnitTestDir = "/.dusk/unittest"

func init() {
	log.SetOutput(ioutil.Discard)
}

func TestConnect(t *testing.T) {
	cm := connmgr.GetInstance()
	cm.Run()

	var c *connmgr.Connmgr
	monkey.PatchInstanceMethod(reflect.TypeOf(c), "OnConnection", func(_ *connmgr.Connmgr, _ net.Conn, _ string) {
		return
	})

	ipport := "google.com:80"

	r := connmgr.Request{Addr: ipport}

	err := cm.Connect(&r)
	assert.Equal(t, nil, err)
	assert.Equal(t, 1, len(cm.ConnectedList))
}

func TestNewRequest(t *testing.T) {

	address := payload.NewNetAddress("216.58.212.174", 80)
	cfg.Set("net.peer.seeds", address.String())

	var c *connmgr.Connmgr
	monkey.PatchInstanceMethod(reflect.TypeOf(c), "OnConnection", func(_ *connmgr.Connmgr, _ net.Conn, _ string) {
		return
	})

	cm := connmgr.New()
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
	cm := connmgr.New()
	_, conn := net.Pipe()
	cm.ConnectedList[address.String()] = &connmgr.Request{Conn: conn, Addr: address.String()}
	cm.Run()
	cm.NewRequest()
	cm.Disconnect(address.String())

	assert.Equal(t, 0, len(cm.ConnectedList))
}
