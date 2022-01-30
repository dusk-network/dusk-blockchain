// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"os/exec"

	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func startProcess(path string, arg ...string) (*os.Process, error) {
	//nolint
	cmd := exec.Command(path, arg...)
	cmd.Env = os.Environ()

	// Redirect both STDOUT and STDERR to this service log file
	cmd.Stdout = log.StandardLogger().Writer()
	cmd.Stderr = log.StandardLogger().Writer()

	log.WithField("path", path).
		Info("start process")

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return cmd.Process, nil
}

// getLastBlockHeight makes an attempt to fetch last block height of a specified node.
func getLastBlockHeight(ctx context.Context, addr string) (uint64, error) {
	// Construct query to fetch block height
	query := "{\"query\" : \"{ blocks (last: 1) { header { height } } }\"}"

	var result map[string]map[string][]map[string]map[string]int
	if err := sendQuery(ctx, query, addr, &result); err != nil {
		return 0, err
	}

	return uint64(result["data"]["blocks"][0]["header"]["height"]), nil
}

func sendQuery(ctx context.Context, query, addr string, result interface{}) error {
	url := "http://" + addr + "/graphql"

	// Construct query to fetch block height

	buf := bytes.Buffer{}
	if _, err := buf.Write([]byte(query)); err != nil {
		return errors.New("invalid query")
	}

	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(ctx)

	//nolint:gosec
	client := http.DefaultClient

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return err
	}

	return nil
}

func createStateClient(ctx context.Context, address string) (rusk.StateClient, *grpc.ClientConn, error) {
	conn, err := grpc.DialContext(ctx, address, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithAuthority("dummy"))
	if err != nil {
		return nil, nil, err
	}

	return rusk.NewStateClient(conn), conn, nil
}
