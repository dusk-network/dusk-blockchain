package client

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// JWTInterceptor takes care of the authentication mechanism
type JWTClientInterceptor struct {
	//http     *http.Client
	token string
	//endpoint string

	// Edward keys
	edPk []byte
	edSk []byte
}

// NewClientInterceptor creates a new UnaryClientInterceptor which handles JWT based
// authentication
func NewClientInterceptor() (*JWTClientInterceptor, error) {
	pk, sk, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return &JWTClientInterceptor{
		edPk: pk,
		edSk: sk,
	}, nil
}

// UnaryClientInterceptor is the client interceptor to create and
// authenticate GRPC calls to the wallet server. This is here for reference and
// test only.
func (jwt *JWTClientInterceptor) UnaryClientInterceptor(
	ctx context.Context,
	method string,
	req interface{},
	reply interface{},
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption) error {
	// we discriminate on the method name since some methods should not have
	// require authentication
	switch method {
	case "Login":
		// create a LoginRequest with the Edward Public Key and the signature
		// of said public key
		// call login, get the access token and save it in memory
		if err := invoker(ctx, method, req, reply, cc, opts...); err != nil {
			return err
		}

		// session := reply.(node.Session)
		// jwt.token = session.AccessToken
		return nil
	case "CloseSession":
		// force expire the session token on the server. It is debatable
		// whether one needs the token to close a session. Here this is the
		// case
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "bearer "+jwt.token)
		return invoker(ctx, method, req, reply, cc, opts...)
	default:
		// rest of the calls use the token acquired with the Login
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "bearer "+jwt.token)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
