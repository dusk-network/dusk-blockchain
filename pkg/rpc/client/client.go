package client

import (
	"context"

	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
	logger "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
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
