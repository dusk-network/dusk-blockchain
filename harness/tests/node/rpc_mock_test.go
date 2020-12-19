// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package node

import (
	"context"
	"fmt"
	"time"

	"github.com/dusk-network/dusk-protobuf/autogen/go/node"
	assert "github.com/stretchr/testify/require"

	"net"
	"testing"

	"google.golang.org/grpc"
)

func TestSayHello(t *testing.T) {
	assert := assert.New(t)
	s := grpc.NewServer()
	node.RegisterWalletServer(s, &node.WalletMock{})

	lis, _ := net.Listen("tcp", "127.0.0.1:5051")
	go func() {
		if err := s.Serve(lis); err != nil {
			panic(fmt.Sprintf("Server exited with error: %v", err))
		}
	}()

	dialCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(dialCtx, "127.0.0.1:5051", grpc.WithInsecure())
	assert.NoError(err)
	defer conn.Close()

	client := node.NewWalletClient(conn)

	callCtx, cancelCtx := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelCtx()

	resp, err := client.GetTxHistory(callCtx, &node.EmptyRequest{})
	assert.NoError(err)

	assert.Equal(8, len(resp.Records))
}
