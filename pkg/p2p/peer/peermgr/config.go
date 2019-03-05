package peermgr

import (
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

// ResponseHandler holds all response handlers related to response meesages from an external peer
type ResponseHandler struct {
	OnHeaders        func(*Peer, *payload.MsgHeaders)
	OnNotFound       func(*Peer, *payload.MsgNotFound)
	OnGetData        func(*Peer, *payload.MsgGetData)
	OnTx             func(*Peer, *payload.MsgTx)
	OnGetHeaders     func(*Peer, *payload.MsgGetHeaders)
	OnAddr           func(*Peer, *payload.MsgAddr)
	OnGetAddr        func(*Peer, *payload.MsgGetAddr)
	OnGetBlocks      func(*Peer, *payload.MsgGetBlocks)
	OnBlock          func(*Peer, *payload.MsgBlock)
	OnConsensus      func(*Peer, *payload.MsgConsensus)
	OnCertificate    func(*Peer, *payload.MsgCertificate)
	OnCertificateReq func(*Peer, *payload.MsgCertificateReq)
	OnMemPool        func(*Peer, *payload.MsgMemPool)
	OnPing           func(*Peer, *payload.MsgPing)
	OnPong           func(*Peer, *payload.MsgPong)
	OnReject         func(*Peer, *payload.MsgReject)
	OnInv            func(*Peer, *payload.MsgInv)
}

type Config struct {
	Magic   protocol.Magic
	Nonce   uint64
	Handler ResponseHandler
}
