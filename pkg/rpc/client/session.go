// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package client

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"time"

	logger "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// NodeClient is a wrapper for the grpc node service client. It takes care of
// handling the session on behalf of the caller by maintaining a grpc
// interceptor.
type NodeClient struct {
	pk ed25519.PublicKey
	sk ed25519.PrivateKey

	addr           string
	proto          string
	persistentConn *grpc.ClientConn

	ctx     context.Context
	cancel  context.CancelFunc
	errChan chan error

	authClient     *AuthClient
	sessionHandler *AuthClientInterceptor
}

// New creates a new NodeClient which takes care of maintaining the session.
func New(proto, addr string) *NodeClient {
	// create the client
	pk, sk, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	nc := &NodeClient{
		sessionHandler: NewClientInterceptor(pk, sk),
		addr:           addr,
		proto:          proto,
		pk:             pk,
		sk:             sk,
		ctx:            ctx,
		cancel:         cancel,
		errChan:        make(chan error, 1),
	}

	go nc.monitorError()

	log.WithFields(logger.Fields{
		"network": proto,
		"address": addr,
	}).Traceln("created grpc session client")
	return nc
}

// PublicKey returns the serialized binary array of the public key of this node
// client.
func (n *NodeClient) PublicKey() []byte {
	return []byte(n.pk)
}

// GetSessionConn triggers a CreateSession through the auth client and
// implicitly populates the interceptor with the session token.
// It returns a new connection or an error.
func (n *NodeClient) GetSessionConn(options ...grpc.DialOption) (*grpc.ClientConn, error) {
	if n.IsSessionActive() {
		if n.persistentConn != nil {
			return n.persistentConn, nil
		}
	}

	// session is not active but we still have a connection dangling around
	if n.persistentConn != nil {
		// first we close the connection
		_ = n.persistentConn.Close()
	}

	// recreating the session and the connection
	if err := n.resetConn(options...); err != nil {
		return nil, err
	}

	n.authClient = NewClient(n.persistentConn, n.pk, n.sk)
	_, err := n.authClient.CreateSession()
	return n.persistentConn, err
}

// ScheduleSessionRefresh refreshes the session with the specified cadence.
func (n *NodeClient) ScheduleSessionRefresh(cadence time.Time, options ...grpc.DialOption) error {
	// GetSessionConn refreshes the authclient if needed
	if _, err := n.GetSessionConn(options...); err != nil {
		return err
	}

	go ScheduleRefreshToken(n.ctx, n.authClient, n.sessionHandler, cadence, 3, n.errChan)
	return nil
}

// DropSession closes a session, invalidate the session token and closes the
// persistent connection gracefully.
func (n *NodeClient) DropSession(options ...grpc.DialOption) error {
	var err error

	defer n.Close()

	if !n.isConnActive() {
		if err = n.resetConn(options...); err != nil {
			return err
		}
	}

	authClient := NewClient(n.persistentConn, n.pk, n.sk)
	err = authClient.DropSession()

	n.sessionHandler.invalidateToken()
	return err
}

func (n *NodeClient) monitorError() {
	err := <-n.errChan
	log.WithError(err).Warnln("problem in session handling, closing the client")
	n.Close()
}

// GracefulClose cancels the session refresh and actively drops the session.
func (n *NodeClient) GracefulClose(opts ...grpc.DialOption) {
	if n.IsSessionActive() {
		// DropSession includes a Close
		_ = n.DropSession(opts...)
		return
	}

	// close the node
	n.Close()
}

// Close the client without notifying the server to close the session.
func (n *NodeClient) Close() {
	n.cancel()
	n.invalidateConn()
}

func (n *NodeClient) isConnActive() bool {
	return n.persistentConn != nil
}

// IsSessionActive returns whether the session is active or otherwise.
func (n *NodeClient) IsSessionActive() bool {
	return n.sessionHandler != nil && n.sessionHandler.isSessionActive()
}

func (n *NodeClient) invalidateConn() {
	n.persistentConn = nil
}

func (n *NodeClient) resetConn(options ...grpc.DialOption) error {
	options = append(
		options,
		grpc.WithContextDialer(getDialer(n.proto)),
		grpc.WithUnaryInterceptor(n.sessionHandler.Unary()),
	)

	// create the GRPC connection
	conn, err := grpc.Dial(
		n.addr,
		options...,
	)
	if err != nil {
		return err
	}

	n.persistentConn = conn
	return nil
}

func getDialer(proto string) func(context.Context, string) (net.Conn, error) {
	d := &net.Dialer{}
	return func(ctx context.Context, addr string) (net.Conn, error) {
		return d.DialContext(ctx, proto, addr)
	}
}
