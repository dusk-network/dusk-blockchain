package server_test

import (
	"context"
	"encoding/json"
	"time"

	"crypto/ed25519"
	"crypto/rand"

	"github.com/dusk-network/dusk-blockchain/pkg/rpc"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/hashset"
	"github.com/dusk-network/dusk-protobuf/autogen/go/node"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type (
	// AuthClient is the client used to test the authorization service
	AuthClient struct {
		service node.AuthClient
		edPk    ed25519.PublicKey
		edSk    ed25519.PrivateKey
	}

	// AuthClientInterceptor handles the authorization for the grpc systems
	AuthClientInterceptor struct {
		client      *AuthClient
		accessToken string
		authMethods *hashset.Set
	}
)

// NewClient returns a new AuthClient
func NewClient(cc *grpc.ClientConn) *AuthClient {
	pk, sk, _ := ed25519.GenerateKey(rand.Reader)
	return &AuthClient{
		service: node.NewAuthClient(cc),
		edPk:    pk,
		edSk:    sk,
	}
}

// NewClientInterceptor creates the client interceptor
func NewClientInterceptor(client *AuthClient, refresh time.Duration) (*AuthClientInterceptor, error) {
	methods := hashset.New()
	methods.Add([]byte("login"))
	interceptor := &AuthClientInterceptor{
		client:      client,
		authMethods: methods,
	}

	err := interceptor.scheduleRefreshToken(refresh)
	if err != nil {
		return nil, err
	}

	return interceptor, nil
}

// CreateSession creates a JWT
func (c *AuthClient) CreateSession() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

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

// DropSession deletes the user PK from the set
func (c *AuthClient) DropSession() error {
	_, err := c.service.DropSession(context.Background(), &node.EmptyRequest{})
	return err
}

// Unary returns the grpc unary interceptor
func (i *AuthClientInterceptor) Unary() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if !i.authMethods.Has([]byte(method)) {
			tky, err := i.attachToken(ctx)
			if err != nil {
				return err
			}
			return invoker(tky, method, req, reply, cc, opts...)
		}

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// attachToken creates the authorization header from a JSON object with the
// following fields:
// - Token: the jwt encoded access token
// - time: the Unix time expressed as an int64
// - signature: the ED25519 signature of the JSON marshaling of the AuthToken
// object (without the signature, obviously)
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
	sig := ed25519.Sign(i.client.edSk, jb)

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

func (i *AuthClientInterceptor) scheduleRefreshToken(refresh time.Duration) error {

	if err := i.refreshToken(); err != nil {
		return err
	}

	go func() {
		wait := refresh
		for {
			time.Sleep(wait)
			if err := i.refreshToken(); err != nil {
				wait = time.Second
				continue
			}

			wait = refresh
		}
	}()

	return nil
}

func (i *AuthClientInterceptor) refreshToken() error {
	accessToken, err := i.client.CreateSession()
	if err != nil {
		return err
	}

	i.accessToken = accessToken
	return nil
}
