// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

//go:build testbed
// +build testbed

package testbed

import (
	"context"
	"encoding/hex"
	"io"
	"os"
	"os/exec"
	"strconv"
	"sync"

	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type testNode struct {
	kadcastPort int
	grpcPort    int
	grpcAddress string

	exe *exec.Cmd

	mu      sync.RWMutex
	message []byte
}

func NewRemotelNode(grpcAddress string, grpcPort int) (*testNode, error) {
	n := &testNode{
		kadcastPort: 0,
		grpcPort:    grpcPort,
		message:     make([]byte, 0),
		grpcAddress: grpcAddress,
	}

	return n, nil
}

func NewLocalNode(exePath string, bootstrapNodes []string, kadcastPort, grpcPort int) (*testNode, error) {
	n := &testNode{
		kadcastPort: kadcastPort,
		grpcPort:    grpcPort,
		message:     make([]byte, 0),
		grpcAddress: "127.0.0.1",
	}

	cmd := exec.Command(exePath,
		"--ipc_method",
		"tcp_ip",
		"-p",
		strconv.Itoa(grpcPort),
		"--kadcast_autobroadcast",
		"--kadcast_public_address",
		NetworkAddr+":"+strconv.Itoa(kadcastPort),
		"--kadcast_bootstrap",
		bootstrapNodes[0],
		"--kadcast_bootstrap",
		bootstrapNodes[1],
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	log.WithField("kadcastPort", kadcastPort).
		WithField("grpc", grpcPort).
		Info("Start a node")

	n.exe = cmd

	return n, nil
}

// GetMessage returns a copy of a received msg.
func (t *testNode) GetMessage() []byte {
	t.mu.RLock()
	cpy := make([]byte, len(t.message))
	copy(cpy, t.message)
	t.mu.RUnlock()

	return cpy
}

// Kill causes the Rusk Process to exit immediately.
func (t *testNode) Kill() {
	if t.exe != nil {
		t.exe.Process.Kill()
	}
}

// Broadcast sends broadcast grpc call from this node.
func (t *testNode) Broadcast(ctx context.Context, payload []byte) error {
	addr := t.grpcAddress + ":" + strconv.Itoa(t.grpcPort)
	conn, _ := createNetworkClient(ctx, addr)

	m := &rusk.BroadcastMessage{
		KadcastHeight: 128,
		Message:       payload,
	}
	// broadcast message
	if _, err := conn.Broadcast(ctx, m); err != nil {
		log.WithError(err).Warn("failed to broadcast message")
		return err
	}

	log.WithField("kadcast_port", t.kadcastPort).WithField("payload", hex.EncodeToString(payload)).
		Info("Broadcast message")

	return nil
}

func createNetworkClient(ctx context.Context, address string) (rusk.NetworkClient, *grpc.ClientConn) {
	conn, err := grpc.DialContext(ctx, address, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.WithField("addr", address).Panic(err)
	}

	return rusk.NewNetworkClient(conn), conn
}

// Listen creates a client connection and accepts stream messages.
func (t *testNode) Listen(ctx context.Context) {
	addr := t.grpcAddress + ":" + strconv.Itoa(t.grpcPort)
	conn, _ := createNetworkClient(ctx, addr)

	// create stream handler
	stream, err := conn.Listen(ctx, &rusk.Null{})
	if err != nil {
		log.WithError(err).Error("open stream error")
		return
	}

	log.WithField("kadcast_port", t.kadcastPort).WithField("grpcAddress", t.grpcAddress).WithField("grpcPort", t.grpcPort).Info("Listening")

	// accept  stream messages
	for {
		msg, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				log.Info("stream terminated")
				return
			}

			log.WithError(err).Warning("recv error")
			return
		}

		// Message received
		log.WithField("kadcast_port", t.kadcastPort).
			WithField("payload", len(msg.Message)).
			WithField("height", msg.Metadata.KadcastHeight).
			WithField("grpcAddress", t.grpcAddress).
			WithField("from", msg.Metadata.SrcAddress).
			Info("received msg")

		t.mu.Lock()
		t.message = make([]byte, len(msg.Message))
		copy(t.message, msg.Message)
		t.mu.Unlock()
	}
}
