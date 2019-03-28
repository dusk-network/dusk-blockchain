// This functionality of the peer should be relocated/restructured once the voucher
// seeder is properly coded. The voucher seeder will take over the responsibility of
// uploading blocks to new nodes, before letting them into the network.

package peer

var (
	// This is the maximum amount of inflight objects that we would like to have
	// Number taken from original codebase
	maxBlockRequest = 1024

	// This is the maximum amount of blocks that we will ask for from a single peer
	// Number taken from original codebase
	maxBlockRequestPerPeer = 16
)

// func (p *Peer) getHeaders(locator, hashStop []byte) (*messages.HeadersMessage, error) {
// 	msg := &messages.HeadersMessage{}
// 	headers := p.chain.GetHeaders(locator, hashStop)

// 	msg.Headers = headers
// 	return msg, nil
// }

// // OnGetHeaders receives 'getheaders' msgs from a peer, reads them from the chain db
// // and sends them to the requesting peer.
// func (p *Peer) sendHeaders(msgBytes *bytes.Buffer) error {
// 	msg := &messages.GetHeadersMessage{}
// 	if err := msg.Decode(msgBytes); err != nil {
// 		return err
// 	}

// 	msgHeaders, err := p.getHeaders(msg.Locator, msg.HashStop)
// 	if err != nil {
// 		return err
// 	}

// 	buffer := new(bytes.Buffer)
// 	if err := msgHeaders.Encode(buffer); err != nil {
// 		return err
// 	}

// 	return p.WriteMessage(buffer, topics.Headers)
// }

// // OnHeaders receives 'headers' msgs from an other peer and adds them to the chain.
// func (p *Peer) onHeaders(msgBytes *bytes.Buffer) error {
// 	msg := &messages.HeadersMessage{}
// 	if err := msg.Decode(msgBytes); err != nil {
// 		return err
// 	}

// 	if len(msg.Headers) == 0 {
// 		return errors.New("headers message contained no headers")
// 	}

// 	return nil
// }

// OnGetData receives 'getdata' msgs from a peer.
// This could be a request for a specific Tx or Block and will be read from the chain db.
// and send to the requesting peer.
// func (p *Peer) onGetData(msgBytes *bytes.Buffer) error {
// 	msg := &messages.GetDataMessage{}

// 	// The caller peer wants some txs and/or blocks from our blockchain.
// 	for _, vector := range msg.Vectors {
// 		switch vector.Type {
// 		case messages.InvTx:
// 			//tx := transactions.NewTX()
// 			//tx.Hash = vector.Hash
// 			//p.chain.GetTx()
// 		case messages.InvBlock:
// 			block := p.getBlock(vector.Hash)
// 			blockMsg := &messages.BlockMessage{block}
// 			buffer := new(bytes.Buffer)
// 			if err := blockMsg.Encode(buffer); err != nil {
// 				return err
// 			}

// 			return p.WriteMessage(buffer, topics.Block)
// 		}
// 	}

// 	return nil
// }

// func (p *Peer) sendBlocks(hashes [][]byte) error {
// 	for _, hash := range hashes {
// 		block := p.chain.GetBlock(hash)
// 		blockMsg := &messages.BlockMessage{block}
// 		buffer := new(bytes.Buffer)
// 		if err := blockMsg.Encode(buffer); err != nil {
// 			return err
// 		}

// 		if err := p.WriteMessage(buffer, topics.Block); err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }

// func (p *Peer) sendAddresses() {
// 	addresses := p.address.GetGoodAddresses()
// 	addrMsg := payload.NewMsgAddr()
// 	for _, addr := range addresses {
// 		addrMsg.AddAddr(&addr)
// 	}
// 	// Push most recent peers to peer
// 	p.Write(addrMsg)
// }
