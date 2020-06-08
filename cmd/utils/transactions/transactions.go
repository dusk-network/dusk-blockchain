package transactions

import (
	"context"
	"errors"
	"time"

	node "github.com/dusk-network/dusk-protobuf/autogen/go/node"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// Transaction is the struct that holds common transaction fields
type Transaction struct {
	Amount   uint64
	LockTime uint64
	TXtype   string
	Address  string
}

// RunTransactions will
func RunTransactions(grpcHost string, transaction Transaction) (*node.TransferResponse, error) {

	log.WithField("transaction", transaction).Info("Set up a connection to the grpc server")
	conn, err := grpc.Dial(grpcHost, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = conn.Close()
	}()

	client := node.NewNodeClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var resp *node.TransferResponse
	switch transaction.TXtype {
	case "consensus":
		log.WithField("transaction", transaction).Info("Sending consensus tx")
		req := node.ConsensusTxRequest{Amount: transaction.Amount, LockTime: transaction.LockTime}
		resp, err = client.SendBid(ctx, &req)
	case "stake":
		log.WithField("transaction", transaction).Info("Sending stake tx")
		req := node.ConsensusTxRequest{Amount: transaction.Amount, LockTime: transaction.LockTime}
		resp, err = client.SendStake(ctx, &req)
	case "transfer":
		log.WithField("transaction", transaction).Info("Sending transfer tx")

		if transaction.Address == "self" {
			log.WithField("transaction", transaction).Debug("Address defined as self, will go get node self address via grpc ...")
			resp, err1 := client.GetAddress(context.Background(), &node.EmptyRequest{})
			if err1 != nil {
				return nil, err1
			}
			log.WithField("address", string(resp.Key.PublicKey)).Info("Sending transfer tx to self address")
			transaction.Address = string(resp.Key.PublicKey)
		}

		req := node.TransferRequest{Amount: transaction.Amount, Address: []byte(transaction.Address)}
		resp, err = client.Transfer(ctx, &req)
	default:
		log.Info("")
		err = errors.New("invalid TXtype")
	}

	if err != nil {
		return nil, err
	}

	return resp, nil
}
