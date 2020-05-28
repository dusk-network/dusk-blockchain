package client

import (
	"context"

	"encoding/json"
	"time"

	"crypto/ed25519"
	"crypto/rand"

	"github.com/dusk-network/dusk-blockchain/pkg/rpc"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/hashset"
	"github.com/dusk-network/dusk-protobuf/autogen/go/node"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
	logger "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var log = logger.WithFields(logger.Fields{"prefix": "grpc"})

// CreateRuskClient opens the connection with the Rusk gRPC server, and
// initializes the different clients which can speak to the Rusk server.
//
// As the Rusk server is a fundamental part of the node functionality,
// this function will panic if the connection can not be established
// successfully.
// FIXME: we need to add the TLS certificates to both ends to encrypt the
// channel.
// QUESTION: should this function be triggered everytime we wanna query
// RUSK or the client can remain connected?
func CreateRuskClient(ctx context.Context, address string) (rusk.RuskClient, *grpc.ClientConn) {
	// NOTE: the grpc client connections should be reused for the lifetime of
	// the client. For this reason, we do not set a timeout at this stage

	// FIXME: create TLS channel here
	conn, err := grpc.DialContext(ctx, address, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Panic(err)
	}

	return rusk.NewRuskClient(conn), conn
}

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

// NewClientInterceptor creates the client interceptor and creates the session. If a duration of 0 is
// specified then the session needs to be created manually
func NewClientInterceptor(client *AuthClient, refresh time.Duration) (*AuthClientInterceptor, error) {
	methods := hashset.New()
	methods.Add([]byte("login"))
	interceptor := &AuthClientInterceptor{
		client:      client,
		authMethods: methods,
	}

	if refresh != 0 {
		err := interceptor.scheduleRefreshToken(refresh)
		if err != nil {
			return nil, err
		}
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
