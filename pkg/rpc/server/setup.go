// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

// This package represents the GRPC server exposing functions to interoperate
// with the node components as well as the wallet
package server

import (
	"os"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-protobuf/autogen/go/node"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var log = logrus.WithField("process", "grpc_s")

// Setup is a configuration struct to setup the GRPC with.
type Setup struct {
	SessionDurationMins uint
	RequireSession      bool
	EnableTLS           bool
	CertFile            string
	KeyFile             string
	Network             string
	Address             string
}

// FromCfg creates a Setup from the configuration. This is handy when a
// configuration should be used (i.e. outside of tests).
func FromCfg() Setup {
	rpc := config.Get().RPC
	return Setup{
		SessionDurationMins: rpc.SessionDurationMins,
		CertFile:            rpc.CertFile,
		KeyFile:             rpc.KeyFile,
		Network:             rpc.Network,
		Address:             rpc.Address,
		RequireSession:      rpc.RequireSession,
	}
}

// SetupGRPC will create a new gRPC server with the correct authentication
// and TLS settings. This server can then be used to register services.
// Note that the server still needs to be turned on (`Serve`).
func SetupGRPC(conf Setup) (*grpc.Server, error) {
	// creating the JWT token manager
	jwtMan, err := NewJWTManager(time.Duration(conf.SessionDurationMins) * time.Minute)
	if err != nil {
		return nil, err
	}

	if conf.Network == "unix" {
		// Remove obsolete unix socket file
		_ = os.Remove(conf.Address)
	}

	// Add default interceptors to provide jwt-based session authentication and error logging
	// for both unary and stream RPC calls
	serverOpt := make([]grpc.ServerOption, 0)

	// Enable TLS if configured
	opt, tlsVer := loadTLSFiles(conf.EnableTLS, conf.CertFile, conf.KeyFile, conf.Network)
	if opt != nil {
		serverOpt = append(serverOpt, opt)
	}

	log.WithField("tls", tlsVer).Info("HTTP server TLS configured")

	grpc.EnableTracing = false

	if conf.RequireSession {
		// instantiate the auth service and the interceptor
		auth, authInterceptor := NewAuth(jwtMan)

		// serverOpt = append(serverOpt, grpc.StreamInterceptor(streamInterceptor))
		serverOpt = append(serverOpt, grpc.UnaryInterceptor(authInterceptor.Unary()))
		grpcServer := grpc.NewServer(serverOpt...)

		// hooking up the Auth service
		node.RegisterAuthServer(grpcServer, auth)
		return grpcServer, nil
	}

	return grpc.NewServer(serverOpt...), nil
}

func loadTLSFiles(enable bool, certFile, keyFile, network string) (grpc.ServerOption, string) {
	tlsVersion := "disabled"

	if !enable {
		if network != "unix" {
			// Running gRPC over tcp would require TLS
			log.Warn("running over insecure HTTP")
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
