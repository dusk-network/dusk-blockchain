package peermgr

import (
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

// ResponseHandler holds all response handlers related to response meesages from an external peer
type ResponseHandler struct {
}

type Config struct {
	Magic   protocol.Magic
	Nonce   uint64
	Handler ResponseHandler
}
