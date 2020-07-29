package server

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"

	"github.com/dusk-network/dusk-blockchain/pkg/rpc"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/hashset"
	"github.com/dusk-network/dusk-protobuf/autogen/go/node"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type authField int

const (
	edPkField authField = iota
)

type (
	// Auth struct is a bit weird since it contains an array of known public keys,
	// while the client should just be one. Oh well :)
	Auth struct {
		store  *hashset.SafeSet
		jwtMan *JWTManager
	}

	// AuthInterceptor is the grpc interceptor to authenticate grpc calls
	// before they get forwarded to the relevant services
	AuthInterceptor struct {
		jwtMan      *JWTManager
		store       *hashset.SafeSet
		openMethods *hashset.Set
	}
)

// NewAuth is the authorization service to manage the session with a client
func NewAuth(j *JWTManager) (*Auth, *AuthInterceptor) {
	safeSet := hashset.NewSafe()
	return &Auth{
			store:  safeSet,
			jwtMan: j,
		}, &AuthInterceptor{
			store:       safeSet,
			jwtMan:      j,
			openMethods: rpc.OpenRoutes,
		}
}

// CreateSession as defined from the grpc service
// Calling createSession from an attached client should refreshes the
// session token (i.e. drop the current one and create a new one)
func (a *Auth) CreateSession(ctx context.Context, req *node.SessionRequest) (*node.Session, error) {
	edPk := req.GetEdPk()
	edSig := req.GetEdSig()

	if !ed25519.Verify(ed25519.PublicKey(edPk), edPk, edSig) {
		return nil, status.Error(codes.Internal, errAccessDenied.Error())
	}

	// delete the session key and recreate one
	encoded := base64.StdEncoding.EncodeToString(edPk)
	token, err := a.jwtMan.Generate(encoded)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot generate token: %v", err)
	}

	// add the PK to the set of known PKs
	_ = a.store.Add(edPk)

	res := &node.Session{AccessToken: token}
	return res, nil
}

// DropSession as defined from the grpc service
func (a *Auth) DropSession(ctx context.Context, req *node.EmptyRequest) (*node.GenericResponse, error) {
	// retrieve client public key from context
	clientPk, ok := ctx.Value(edPkField).([]byte)
	if !ok {
		return nil, status.Error(codes.Internal, "unable to retrieve client pk from context")
	}
	// remove the PK to the set of known PKs
	_ = a.store.Remove(clientPk)

	res := &node.GenericResponse{Response: "session successfully dropped"}
	return res, nil
}

// Unary returns a UnaryServerInterceptor responsible for authentication
func (ai *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		tag := "Unary call " + info.FullMethod
		log.Tracef("%s", tag)

		vctx, err := ai.authorize(ctx, info.FullMethod)
		if err != nil {
			return nil, err
		}

		return handler(vctx, req)
	}
}

func (ai *AuthInterceptor) authorize(ctx context.Context, method string) (context.Context, error) {
	if ai.openMethods.Has([]byte(method)) {
		return ctx, nil
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx, status.Errorf(codes.Unauthenticated, "metadata not provided")
	}

	values := md["authorization"]
	if len(values) == 0 {
		return ctx, status.Error(codes.Unauthenticated, "token not provided")
	}

	clientPk, err := ai.extractClientPK(values[0])
	if err != nil {
		return ctx, status.Errorf(codes.Unauthenticated, "error in extracting the client PK: %v", err)
	}

	return context.WithValue(ctx, edPkField, clientPk), nil
}

func (ai *AuthInterceptor) extractClientPK(a string) ([]byte, error) {
	authToken := &rpc.AuthToken{}
	// unmarshaling the authToken in the authentication header field
	if err := json.Unmarshal([]byte(a), authToken); err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "could not unmarshal auth token struct: %v", err)
	}

	// verify the JWT session token
	claims, err := ai.jwtMan.Verify(authToken.AccessToken)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid access token: %v", err)
	}

	// extract the edPK of the client
	b64EdPk := claims.ClientEdPk
	edPk, err := base64.StdEncoding.DecodeString(b64EdPk)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not decode sender")
	}

	if !ai.store.Has(edPk) {
		return nil, status.Errorf(codes.Internal, "client does not have an active session")
	}

	// verify the client signature with extracted public key
	if !authToken.Verify(edPk) {
		return nil, status.Error(codes.Internal, "error in signature verification")
	}

	return edPk, nil
}
