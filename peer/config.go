package peer

import (
	"gitlab.dusk.network/dusk-core/dusk-go/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/wire/protocol"
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
	StartHeight  func() uint32
	OnHeader     func(*Peer, *payload.MsgHeaders)
	OnGetHeaders func(msg *payload.MsgGetHeaders) // returns HeaderMessage
	OnAddr       func(*Peer, *payload.MsgAddr)
	OnGetAddr    func(*Peer, *payload.MsgGetAddr)
	OnInv        func(*Peer, *payload.MsgInv)
	OnGetData    func(msg *payload.MsgGetData)
	OnBlock      func(*Peer, *payload.MsgBlock)
	OnGetBlocks  func(msg *payload.MsgGetBlocks)
}
