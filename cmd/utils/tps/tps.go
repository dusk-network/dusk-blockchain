// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package tps

import (
	"context"
	"time"

	"github.com/dusk-network/dusk-protobuf/autogen/go/node"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// StartSpamming transactions from the given GRPC address. The utility will spam
// Transfer transactions right back to the wallet it's coming from.
func StartSpamming(addr string, delay int, amount uint64) error {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return err
	}

	defer func() {
		if err = conn.Close(); err != nil {
			logrus.WithError(err).Error("error closing connection")
		}
	}()

	client := node.NewTransactorClient(conn)
	delayTime := time.Duration(delay) * time.Millisecond

	walletAddr, err := getWalletAddress(conn)
	if err != nil {
		logrus.WithError(err).Error("could not get wallet address")
		return err
	}

	for {
		time.Sleep(delayTime)

		logrus.WithFields(logrus.Fields{
			"amount": amount,
			"addr":   string(walletAddr),
		}).Debug("sending transfer transaction")

		if err := sendTransaction(amount, client, walletAddr); err != nil {
			logrus.WithError(err).Error("error encountered when sending transaction")
			return err
		}
	}
}

func getWalletAddress(conn *grpc.ClientConn) ([]byte, error) {
	wclient := node.NewWalletClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resp, err := wclient.GetAddress(ctx, &node.EmptyRequest{})
	if err != nil {
		return nil, err
	}

	return resp.Key.PublicKey, nil
}

func sendTransaction(amount uint64, client node.TransactorClient, walletAddr []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := node.TransferRequest{Amount: amount, Address: walletAddr}
	_, err := client.Transfer(ctx, &req)
	return err
}
