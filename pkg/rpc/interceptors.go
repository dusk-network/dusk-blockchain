package rpc

import (
	"context"
	"errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var (
	errAccessDenied    = errors.New("access denied")
	errMissingMetadata = errors.New("missing metadata")
)

var token string

// HTTP Basic authentication per each method call
func authenticate(ctx context.Context, token string) error {

	if len(token) == 0 {
		return nil
	}

	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if len(md["authorization"]) > 0 {
			val := "Basic " + token
			if md["authorization"][0] == val {
				return nil
			}
		}
		return errAccessDenied
	}

	return errMissingMetadata
}

// Stream RPCs - the client sends a request to the server and gets a stream to read a sequence of messages back
func streamInterceptor(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {

	tag := "Stream call " + info.FullMethod
	log.Tracef("%s", tag)

	err := streamInterceptorHandler(srv, stream, info, handler)
	if err != nil {
		log.WithError(err).Errorf("%s failed", tag)
	}

	log.Tracef("%s handled", tag)
	return err
}

// Unary RPCs - client sends a single request to the server and gets a single response back
func unaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

	tag := "Unary call " + info.FullMethod
	log.Tracef("%s", tag)

	res, err := unaryInterceptorHandler(ctx, req, info, handler)
	if err != nil {
		log.WithError(err).Errorf("%s failed", tag)
	}

	log.Tracef("%s handled", tag)
	return res, err
}

//nolint:unparam
func streamInterceptorHandler(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {

	if err := authenticate(stream.Context(), token); err != nil {
		return err
	}

	return handler(srv, stream)
}

//nolint:unparam
func unaryInterceptorHandler(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

	if err := authenticate(ctx, token); err != nil {
		return nil, err
	}

	return handler(ctx, req)
}
