package connmgr

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/util"
	"net"
	"sync"
)

var instance *Connmgr
var once sync.Once

// GetInstance creates a Connmgr instance as a Singleton
func GetInstance() *Connmgr {

	if instance == nil {

		once.Do(func() {
			instance = New()
		})
		loop()
	}

	return instance
}

func loop() {

	go func() {

		ip, err := util.GetOutboundIP()
		if err != nil {
			log.WithField("prefix", "connmgr").Error("Failed to get local ip", err)
		}

		addrPort := ip.String() + ":" + viper.GetString("net.peer.port")
		listener, err := net.Listen("tcp", addrPort)

		if err != nil {
			log.WithField("prefix", "connmgr").Error("Failed to connect to outbound", err)
		}

		defer func() {
			listener.Close()
		}()

		for {
			conn, err := listener.Accept()

			if err != nil {
				continue
			}
			// TODO(kev): in the OnAccept the connection address will be added to AddrMgr
			go instance.OnAccept(conn)
		}

	}()

}
