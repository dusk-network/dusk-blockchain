// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package main

import (
	"context"

	"github.com/dusk-network/dusk-protobuf/autogen/go/node"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// configService implements handlers for node.ConfigService calls.
// The service is about modifying specific config settings in runtime.
// supported configs: logger.level.
type configService struct{}

func newConfigService(srv *grpc.Server) *configService {
	cs := new(configService)

	if srv != nil {
		node.RegisterConfigServer(srv, cs)
	}
	return cs
}

func (cs *configService) ChangeLogLevel(ctx context.Context, c *node.LogLevelRequest) (*node.GenericResponse, error) {
	level, err := logrus.ParseLevel(c.Level)
	if err == nil {
		logrus.SetLevel(level)
	}
	return &node.GenericResponse{}, err
}
