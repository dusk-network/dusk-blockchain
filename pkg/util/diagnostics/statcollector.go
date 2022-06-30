// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package diagnostics

import (
	"bytes"
	"encoding/hex"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	j "encoding/json"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-crypto/hash"
)

var _, enableCollecting = os.LookupEnv("DUSK_ENABLE_TPS_TEST")

const (
	// serviceURL is address of the external network-data-collector.
	serviceURL = "http://localhost:1337/rpc"
)

// RegisterWireMsg is a util for registering a wire message to a network-monitor.
// service in a zero-overhead manner. Under the hood, it encodes a json-rpc with following fields
// msg_id,
// node_address (or node id),
// receive timestamp in,
// networkType is the type of the network (Kadcst, Gossip are allowed valued),
// rawdata is the wire message data.
func RegisterWireMsg(networkType string, rawdata []byte) {
	// Record receive time before running the go-routine sender
	recv_at := time.Now().UnixNano()

	if !enableCollecting {
		return
	}

	go func(recv_at int64) {
		// extract message wire type from the message blob
		buf := bytes.NewBuffer(rawdata)

		category, err := topics.Extract(buf)
		if err != nil {
			return
		}

		// Check if this wire message type should be registered
		if !canRegister(category) {
			return
		}

		digest, err := hash.Xxhash(rawdata)
		if err != nil {
			panic(err)
		}

		// notify monitoring
		// addr := cfg.Get().Network.Port
		addr := "unknown"

		msgID := networkType + "_" + category.String() + "_" + hex.EncodeToString(digest)

		sendNote(msgID, addr, uint64(recv_at))
	}(recv_at)
}

// NoteArgs rpc request args.
type NoteArgs struct {
	MsgID, Addr, MsgType string
	Recv_at              uint64
}

// NoteRequest rpc request.
type NoteRequest struct {
	Method string     `json:"method"`
	Params []NoteArgs `json:"params"`
	ID     string     `json:"id"`
}

// Say rpc method.
func (h *NoteRequest) Say(p NoteArgs) *NoteRequest {
	h.Method = "NoteService.Say"
	h.Params = []NoteArgs{p}
	h.ID = "1"

	return h
}

func sendNote(msgID, addr string, recv_at uint64) {
	r := &NoteRequest{}
	if b, err := j.Marshal(r.Say(NoteArgs{MsgID: msgID, Addr: addr, Recv_at: recv_at})); err != nil {
		log.Fatal(err)
	} else {
		log.Printf("json %s", b)
		if res, err := http.Post(serviceURL, "application/json;charset=UTF-8", strings.NewReader(string(b))); err != nil {
			log.Fatal(err)
		} else {
			// log.Printf("res : %v", res)
			_, err := ioutil.ReadAll(res.Body)
			_ = res.Body.Close()
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

func canRegister(category topics.Topic) bool {
	switch category {
	case topics.Tx:
		// /topics.Block:
		return true
	}

	return false
}
