// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package main

import (
	"context"
	"os"

	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
)

// Service wraps up service process data and methods.
type Service struct {
	Process *os.Process
	Addr    string
	Name    string

	retry uint

	PingFunc func(ctx context.Context, addr string) error
}

// PingDusk ensures the DUSK service is responsive.
func PingDusk(ctx context.Context, addr string) error {
	_, err := getLastBlockHeight(ctx, addr)
	// TODO: Send an alarm if height does not change for more than a minute.
	return err
}

// PingRusk ensures the RUSK service is responsive.
func PingRusk(ctx context.Context, addr string) error {
	// TODO: Add Rusk Ping API
	c, _, err := createStateClient(ctx, addr)
	if err != nil {
		return err
	}

	_, err = c.GetProvisioners(ctx, &rusk.GetProvisionersRequest{})
	return err
}
