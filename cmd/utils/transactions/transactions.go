package transactions

import (
	"context"
	"errors"
	"time"

	pb "github.com/dusk-network/dusk-protobuf/autogen/go/node"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type Transaction struct {
	Amount   uint64
	LockTime uint64
	TXtype   string
	Address  string
}

// RunTransactions will
func RunTransactions(grpcHost string, transaction Transaction) (*pb.TransferResponse, error) {

	log.WithField("transaction", transaction).Info("Set up a connection to the grpc server")
	conn, err := grpc.Dial(grpcHost, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = conn.Close()
	}()

	client := pb.NewNodeClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var resp *pb.TransferResponse
	switch transaction.TXtype {
	case "consensus":
		log.WithField("transaction", transaction).Info("Sending consensus tx")
		req := pb.ConsensusTxRequest{Amount: transaction.Amount, LockTime: transaction.LockTime}
		resp, err = client.SendBid(ctx, &req)
	case "stake":
		log.WithField("transaction", transaction).Info("Sending stake tx")
		req := pb.ConsensusTxRequest{Amount: transaction.Amount, LockTime: transaction.LockTime}
		resp, err = client.SendStake(ctx, &req)
	case "transfer":
		log.WithField("transaction", transaction).Info("Sending transfer tx")
		req := pb.TransferRequest{Amount: transaction.Amount, Address: []byte(transaction.Address)}
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
