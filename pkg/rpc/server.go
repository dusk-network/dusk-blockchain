// This package represents the GRPC server exposing functions to interoperate
// with the node components as well as the wallet
package rpc

import (
	"encoding/base64"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// SetupgRPCServer will create a new gRPC server with the correct authentication
// and TLS settings. This server can then be used to register services.
// Note that the server still needs to be turned on (`Serve`).
func SetupgRPCServer() (*grpc.Server, error) {
	conf := config.Get().RPC

	// Build basic auth token, if configured
	if len(conf.User) > 0 && len(conf.Pass) > 0 {
		msg := conf.User + ":" + conf.Pass
		token = base64.StdEncoding.EncodeToString([]byte(msg))
	} else {
		if conf.Network != "unix" {
			log.Panicf("basic auth is disabled on %s network", conf.Network)
		}
	}

	// Add default interceptors to provide basic authentication and error logging
	// for both unary and stream RPC calls
	serverOpt := make([]grpc.ServerOption, 0)
	serverOpt = append(serverOpt, grpc.StreamInterceptor(streamInterceptor))
	serverOpt = append(serverOpt, grpc.UnaryInterceptor(unaryInterceptor))

	// Enable TLS if configured
	opt, tlsVer := loadTLSFiles(conf.EnableTLS, conf.CertFile, conf.KeyFile, conf.Network)
	if opt != nil {
		serverOpt = append(serverOpt, opt)
	}
	log.WithField("tls", tlsVer).Infof("gRPC HTTP server TLS configured")

	grpcServer := grpc.NewServer(serverOpt...)
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
