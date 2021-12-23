// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package tests

import (
	"fmt"
	"net"

	"google.golang.org/grpc"
)

// StartMockServer will start a mock server.
func StartMockServer(address string) {
	s := grpc.NewServer()

	lis, _ := net.Listen("tcp", address)

	go func() {
		if err := s.Serve(lis); err != nil {
			panic(fmt.Sprintf("Server exited with error: %v", err))
		}
	}()
}
