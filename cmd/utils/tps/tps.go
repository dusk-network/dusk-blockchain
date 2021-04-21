// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package tps

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/dusk-network/dusk-protobuf/autogen/go/node"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const (
	retryAmount   = 5
	retryDelay    = 5 * time.Second
	defaultAmount = 100
)

// StartSpamming transactions from the given GRPC address. The utility will spam
// Transfer transactions right back to the wallet it's coming from.
func StartSpamming(addr string, delay int, amount uint64) error {
	if addr == "" {
		return errors.New("no GRPC address provided")
	}

	// Add UNIX prefix in case we're using unix sockets.
	if strings.Contains(addr, ".sock") {
		addr = "unix://" + addr
	}

	if amount == 0 {
		amount = defaultAmount
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, addr, grpc.WithInsecure(), grpc.WithBlock())
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

		// We retry whenever the first couple of attempts fail. This is due to
		// a failure possibly being related to us spending all of our outputs
		// before the new block is finalized, so a little bit of a wait could
		// fix the issue. If we can't manage to succeed after 5 tries, we return.
		for i := 0; i < retryAmount; i++ {
			logrus.WithFields(logrus.Fields{
				"amount": amount,
				"addr":   string(walletAddr),
			}).Debug("sending transfer transaction")

			err = sendTransaction(amount, client, walletAddr)
			if err == nil {
				break
			}

			if i == retryAmount-1 {
				logrus.WithError(err).Error("didn't succeed in sending transaction")
				return err
			}

			logrus.WithError(err).
				Error("error encountered when sending transaction - retrying...")

			time.Sleep(retryDelay * time.Duration(i+1))
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
