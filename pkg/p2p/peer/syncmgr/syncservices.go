package syncmgr

import (
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
)

func (s *Syncmgr) getHeaders(msg *payload.MsgGetHeaders) (*payload.MsgHeaders, error) {
	var msgheaders payload.MsgHeaders
	locator := msg.Locator
	hashStop := msg.HashStop

	headers, err := s.config.GetHeaders(locator, hashStop)
	if err != nil {
		return nil, err
	}

	msgheaders.Headers = headers
	return &msgheaders, nil
}
