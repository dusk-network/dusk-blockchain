package responding

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
)

type CandidateBroker struct {
	rpcBus       rpcbus.RPCBus
	responseChan chan<- *bytes.Buffer
}
