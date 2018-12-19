package peer

import (
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

// LocalConfig specifies the properties that should be available for each remote peer

type LocalConfig struct {
	Net         protocol.DuskNetwork
	UserAgent   string
	Services    protocol.ServiceFlag
	Nonce       uint32
	ProtocolVer uint32
	Relay       bool
	Port        uint16
	// Pointer to config will keep the startheight updated for each MsgVersion we plan to send
	StartHeight      func() uint32
	OnHeader         func(*Peer, *payload.MsgHeaders)
	OnGetHeaders     func(*payload.MsgGetHeaders) // returns HeaderMessage
	OnAddr           func(*Peer, *payload.MsgAddr)
	OnGetAddr        func(*Peer, *payload.MsgGetAddr)
	OnInv            func(*Peer, *payload.MsgInv)
	OnGetData        func(*payload.MsgGetData)
	OnBlock          func(*Peer, *payload.MsgBlock)
	OnGetBlocks      func(*payload.MsgGetBlocks)
	OnBinary         func(*Peer, *payload.MsgBinary)
	OnCandidate      func(*Peer, *payload.MsgCandidate)
	OnCertificate    func(*Peer, *payload.MsgCertificate)
	OnCertificateReq func(*Peer, *payload.MsgCertificateReq)
	OnMemPool        func(*Peer, *payload.MsgMemPool)
	OnNotFound       func(*Peer, *payload.MsgNotFound)
	OnPing           func(*Peer, *payload.MsgPing)
	OnPong           func(*payload.MsgPong)
	OnReduction      func(*Peer, *payload.MsgReduction)
	OnReject         func(*Peer, *payload.MsgReject)
	OnScore          func(*Peer, *payload.MsgScore)
}
