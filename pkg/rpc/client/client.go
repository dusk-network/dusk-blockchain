// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package client

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/rpc"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/hashset"
	"github.com/dusk-network/dusk-protobuf/autogen/go/node"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
	logger "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var log = logger.WithFields(logger.Fields{"process": "grpc_c"})

// CreateStateClient opens the connection with the Rusk gRPC server, and
// initializes the different clients which can speak to the Rusk server.
//
// As the Rusk server is a fundamental part of the node functionality,
// this function will panic if the connection can not be established
// successfully.
// FIXME: we need to add the TLS certificates to both ends to encrypt the
// channel.
// QUESTION: should this function be triggered everytime we wanna query
// RUSK or the client can remain connected?
// NOTE: the grpc client connections should be reused for the lifetime of
// the client. For this reason, we do not set a timeout at this stage.
func CreateStateClient(ctx context.Context, address string) (rusk.StateClient, *grpc.ClientConn) {
	// FIXME: create TLS channel here
	conn, err := grpc.DialContext(ctx, address, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Panic(err)
	}

	return rusk.NewStateClient(conn), conn
}

// CreateKeysClient creates a client for the Keys service.
func CreateKeysClient(ctx context.Context, address string) (rusk.KeysClient, *grpc.ClientConn) {
	conn, err := grpc.DialContext(ctx, address, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Panic(err)
	}

	return rusk.NewKeysClient(conn), conn
}

// CreateTransferClient creates a client for the Transfer service.
func CreateTransferClient(ctx context.Context, address string) (rusk.TransferClient, *grpc.ClientConn) {
	conn, err := grpc.DialContext(ctx, address, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Panic(err)
	}

	return rusk.NewTransferClient(conn), conn
}

// CreateStakeClient creates a client for the Stake service.
func CreateStakeClient(ctx context.Context, address string) (rusk.StakeServiceClient, *grpc.ClientConn) {
	conn, err := grpc.DialContext(ctx, address, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Panic(err)
	}

	return rusk.NewStakeServiceClient(conn), conn
}

type (
	// AuthClient is the client used to test the authorization service.
	AuthClient struct {
		service node.AuthClient
		edPk    ed25519.PublicKey
		edSk    ed25519.PrivateKey
	}

	// AuthClientInterceptor handles the authorization for the grpc systems.
	AuthClientInterceptor struct {
		lock        sync.RWMutex
		accessToken string
		openMethods *hashset.Set
		edPk        ed25519.PublicKey
		edSk        ed25519.PrivateKey
	}
)

// NewClient returns a new AuthClient and AuthClientInterceptor.
func NewClient(cc *grpc.ClientConn, edPk ed25519.PublicKey, edSk ed25519.PrivateKey) *AuthClient {
	return &AuthClient{
		service: node.NewAuthClient(cc),
		edPk:    edPk,
		edSk:    edSk,
	}
}

// NewClientInterceptor injects the session to the outgoing requests.
func NewClientInterceptor(edPk ed25519.PublicKey, edSk ed25519.PrivateKey) *AuthClientInterceptor {
	return &AuthClientInterceptor{
		openMethods: rpc.OpenRoutes,
		edSk:        edSk,
		edPk:        edPk,
	}
}

// CreateSession creates a JWT.
func (c *AuthClient) CreateSession() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return c.createSessionWithContext(ctx)
}

func (c *AuthClient) createSessionWithContext(ctx context.Context) (string, error) {
	edSig := ed25519.Sign(c.edSk, c.edPk)
	req := &node.SessionRequest{
		EdPk:  c.edPk,
		EdSig: edSig,
	}

	res, err := c.service.CreateSession(ctx, req)
	if err != nil {
		return "", err
	}

	return res.GetAccessToken(), nil
}

// DropSession deletes the user PK from the set.
func (c *AuthClient) DropSession() error {
	_, err := c.service.DropSession(context.Background(), &node.EmptyRequest{})
	return err
}

func (i *AuthClientInterceptor) isSessionActive() bool {
	i.lock.RLock()
	defer i.lock.RUnlock()
	return i.accessToken != ""
}

// Unary returns the grpc unary interceptor. It implictly saves the access
// token when a new session is created and drops it when the session gets
// explicitly dropped.
func (i *AuthClientInterceptor) Unary() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if !i.openMethods.Has([]byte(method)) {
			tky, err := i.attachToken(ctx)
			if err != nil {
				return err
			}

			err = invoker(tky, method, req, reply, cc, opts...)

			// following an explicit session drop, the session token is
			// invalidated, no matter the error
			if method == rpc.DropSessionRoute {
				i.invalidateToken()
			}

			return err
		}

		if method == rpc.CreateSessionRoute {
			err := invoker(ctx, method, req, reply, cc, opts...)
			if err != nil {
				return err
			}

			session := reply.(*node.Session)

			i.SetAccessToken(session.GetAccessToken())
			return nil
		}

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// SetAccessToken sets the session token in a threadsafe way.
func (i *AuthClientInterceptor) SetAccessToken(accessToken string) {
	i.lock.Lock()
	defer i.lock.Unlock()
	i.accessToken = accessToken
}

func (i *AuthClientInterceptor) invalidateToken() {
	i.lock.Lock()
	defer i.lock.Unlock()
	i.accessToken = ""
}

// attachToken creates the authorization header from a JSON object with the
// following fields:
// - Token: the jwt encoded access token
// - time: the Unix time expressed as an int64
// - signature: the ED25519 signature of the JSON marshaling of the AuthToken
// object (without the signature, obviously).
func (i *AuthClientInterceptor) attachToken(ctx context.Context) (context.Context, error) {
	auth := rpc.AuthToken{
		AccessToken: i.accessToken,
		Time:        time.Now().Unix(),
	}

	// first we marshal the AuthToken without the signature
	jb, err := auth.AsSignable()
	if err != nil {
		return ctx, err
	}

	// creating the signature
	sig := ed25519.Sign(i.edSk, jb)

	// adding it to the AuthToken
	auth.Sig = sig

	// json-marshaling the final version
	payload, err := json.Marshal(auth)
	if err != nil {
		return ctx, err
	}

	// adding it to the authorization header
	return metadata.AppendToOutgoingContext(ctx, "authorization", string(payload)), nil
}

// ScheduleRefreshToken is supposed to run in a goroutine. It refreshes the
// session token according to the specified refresh frequency.
func ScheduleRefreshToken(ctx context.Context, client *AuthClient, interceptor *AuthClientInterceptor, refresh time.Time, retries int, errChan chan error) {
	for {
		refreshCtx, cancel := context.WithDeadline(ctx, refresh)
		select {
		case <-ctx.Done(): // parent context cancellation
			cancel()
			return
		case <-refreshCtx.Done(): // time to recreate the Session
			cancel() // avoiding goroutine leaks

			sessionToken, err := refreshToken(ctx, client, 1*time.Second, retries)
			if err != nil {
				errChan <- err
				return
			}

			interceptor.lock.Lock()
			interceptor.accessToken = sessionToken
			interceptor.lock.Unlock()
		}
	}
}

func refreshToken(ctx context.Context, client *AuthClient, timeout time.Duration, retries int) (string, error) {
	for i := 0; i < retries; i++ {
		// first check for context done
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		// creating the timeout
		reqCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		// we use a circuit-breakers, i.e. a goroutine that would invoke
		// CreateSession. The advantage is that we can timeout the request with
		// the context if it takes too long
		sessionChan := make(chan string, 1)
		iErrChan := make(chan error, 1)

		go func() {
			session, err := client.createSessionWithContext(reqCtx)
			if err != nil {
				iErrChan <- err
				return
			}

			sessionChan <- session
		}()

		select {
		case accessToken := <-sessionChan:
			return accessToken, nil
		case err := <-iErrChan:
			return "", err
		case <-reqCtx.Done():
		}

		// sleeping before retrying. Here we should implement exponential backoffs
		time.Sleep(1 * time.Second)
	}

	return "", errors.New("max retries exceeded")
}
