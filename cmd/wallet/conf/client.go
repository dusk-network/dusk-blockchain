// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package conf

import (
	"context"
	"encoding/base64"
	"time"

	"github.com/dusk-network/dusk-protobuf/autogen/go/node"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// NodeClient holds node related fields.
type NodeClient struct {
	dialTimeout       int64
	WalletClient      node.WalletClient
	TransactorClient  node.TransactorClient
	ProvisionerClient node.ProvisionerClient
	ChainClient       node.ChainClient
	MempoolClient     node.MempoolClient
	conn              *grpc.ClientConn
}

// NewNodeClient returns a nodeClient with fixed dialTimeout of 5s.
func NewNodeClient() *NodeClient {
	return &NodeClient{
		dialTimeout: 5,
	}
}

// Connect initializes a grpcClient to dusk-blockchain node grpc interface.
// For over-tcp communication, it could enable TLS and Basic Authentication.
func (c *NodeClient) Connect(conf rpcConfiguration) error {
	addr := conf.Address
	if conf.Network == "unix" {
		addr = "unix://" + conf.Address
	}

	dialOptions := make([]grpc.DialOption, 0)
	dialOptions = append(dialOptions, grpc.WithBlock())

	// Create TLS based credential.
	if len(conf.CertFile) > 0 {
		transportCred, err := credentials.NewClientTLSFromFile(conf.CertFile, conf.Hostname)
		if err != nil {
			return err
		}

		credsOpt := grpc.WithTransportCredentials(transportCred)
		dialOptions = append(dialOptions, credsOpt)
	} else {
		if conf.Network != "unix" {
			// Insecure connection can be suitable only for unix socket
			// transport where node and wallet-cli are co-deployed
			panic("insecure transport not allowed over tcp transport")
		}

		dialOptions = append(dialOptions, grpc.WithInsecure())
	}

	// Init dial timeout
	var dialCtx context.Context

	if c.dialTimeout > 0 {
		var cancel context.CancelFunc

		dialCtx, cancel = context.WithTimeout(context.Background(),
			time.Duration(c.dialTimeout)*time.Second)
		defer cancel()
	}

	// Initialize Basic Auth.
	// It requires secured transport by default.
	if len(conf.User) > 0 {
		authOpt := grpc.WithPerRPCCredentials(basicAuth{
			username: conf.User,
			password: conf.Pass,
			secured:  true,
		})

		dialOptions = append(dialOptions, authOpt)
	}

	// Set up a connection to the server.
	conn, err := grpc.DialContext(dialCtx, addr, dialOptions...)
	if err != nil {
		return err
	}

	c.conn = conn
	c.WalletClient = node.NewWalletClient(conn)
	c.TransactorClient = node.NewTransactorClient(conn)
	c.ProvisionerClient = node.NewProvisionerClient(conn)
	c.ChainClient = node.NewChainClient(conn)
	c.MempoolClient = node.NewMempoolClient(conn)

	return nil
}

// Close conn.
func (c *NodeClient) Close() {
	if c.conn != nil {
		_ = c.conn.Close()
	}
}

// Ping not yet implemented.
func (c *NodeClient) Ping() error {
	// TODO:
	return nil
}

// basicAuth builds request metadata to provide HTTP Basic Authentication params.
type basicAuth struct {
	username string
	password string
	secured  bool
}

func (b basicAuth) GetRequestMetadata(ctx context.Context, in ...string) (map[string]string, error) {
	auth := b.username + ":" + b.password
	enc := base64.StdEncoding.EncodeToString([]byte(auth))

	return map[string]string{
		"authorization": "Basic " + enc,
	}, nil
}

func (b basicAuth) RequireTransportSecurity() bool {
	return b.secured
}
