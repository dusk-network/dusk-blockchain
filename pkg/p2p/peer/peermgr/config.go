package peermgr

import (
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

type Config struct {
	Magic protocol.Magic
	Nonce uint64
}
