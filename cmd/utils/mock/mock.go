// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package mock

import (
	"fmt"
	"net"

	"github.com/dusk-network/dusk-protobuf/autogen/go/node"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// RunMock will run a grpc mock for rusk wallet and transactor.
func RunMock(grpcMockHost string) error {
	s := grpc.NewServer()
	node.RegisterTransactorServer(s, &node.TransactorMock{})

	log.WithField("grpcMockHost", grpcMockHost).
		Info("RunMock Action starting ...")

	lis, _ := net.Listen("tcp", grpcMockHost)
	if err := s.Serve(lis); err != nil {
		return fmt.Errorf("server exited with error: %v", err)
	}

	return nil
}
