// This package represents the GRPC server exposing functions to interoperate
// with the node components as well as the wallet
package server

import (
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-protobuf/autogen/go/node"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var log = logrus.WithField("process", "grpc-server")

// SetupGRPCServer will create a new gRPC server with the correct authentication
// and TLS settings. This server can then be used to register services.
// Note that the server still needs to be turned on (`Serve`).
func SetupGRPCServer() (*grpc.Server, error) {
	conf := config.Get().RPC

	// creating the JWT token manager
	jwtMan, err := NewJWTManager(time.Duration(conf.SessionDurationMins) * time.Minute)
	if err != nil {
		return nil, err
	}

	// instantiate the auth service and the interceptor
	auth, authInterceptor := NewAuth(jwtMan)

	// Add default interceptors to provide jwt-based session authentication and error logging
	// for both unary and stream RPC calls
	serverOpt := make([]grpc.ServerOption, 0)
	//serverOpt = append(serverOpt, grpc.StreamInterceptor(streamInterceptor))
	serverOpt = append(serverOpt, grpc.UnaryInterceptor(authInterceptor.Unary()))

	// Enable TLS if configured
	opt, tlsVer := loadTLSFiles(conf.EnableTLS, conf.CertFile, conf.KeyFile, conf.Network)
	if opt != nil {
		serverOpt = append(serverOpt, opt)
	}
	log.WithField("tls", tlsVer).Infof("gRPC HTTP server TLS configured")

	grpcServer := grpc.NewServer(serverOpt...)

	// hooking up the Auth service
	node.RegisterAuthServer(grpcServer, auth)

	grpc.EnableTracing = false
	return grpcServer, nil
}

func loadTLSFiles(enable bool, certFile, keyFile, network string) (grpc.ServerOption, string) {
	tlsVersion := "disabled"
	if !enable {
		if network != "unix" {
			// Running gRPC over tcp would require TLS
			log.Warn("Running over insecure HTTP")
		}

		return nil, tlsVersion
	}

	creds, err := credentials.NewServerTLSFromFile(certFile, keyFile)
	if err != nil {
		// If TLS is explicitly enabled, any error here should cause panic
		log.WithError(err).Panic("could not enable TLS")
	}

	i := creds.Info()
	if i.SecurityProtocol == "ssl" {
		log.WithError(err).Panic("SSL is insecure")
	}

	recommendedVer := "1.3"
	if i.SecurityVersion != recommendedVer { //nolint
		log.Warnf("Recommended TLS version is %s", recommendedVer)
	}

	tlsVersion = i.SecurityVersion //nolint
	return grpc.Creds(creds), tlsVersion
}
