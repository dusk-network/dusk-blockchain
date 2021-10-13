// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package ruskmock

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/chain"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/util"
	"github.com/dusk-network/dusk-blockchain/pkg/util/legacy"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var log = logrus.WithField("process", "mock rusk server")

const (
	stateTransitionDelay = 1 * time.Second
	stakeGracePeriod     = 10
)

// Server is a stand-in Rusk server, which can be used during any kind of
// testing. Its behavior can be modified depending on the settings of the
// `Config` struct, contained in the `Server`, to simulate different types
// of scenarios on demand.
type Server struct {
	cfg *Config
	s   *grpc.Server

	p      *user.Provisioners
	height uint64

	db *BuntStore
}

// New returns a new Rusk mock server with the given config. If no config is
// passed, a default one is put into place.
func New(cfg *Config, c config.Registry) (*Server, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	dbPath := filepath.Dir(filepath.Dir(c.Wallet.Store)) + "/" + "ruskmock.db"

	db, err := NewBuntStore(dbPath, High)
	if err != nil {
		panic(err)
	}

	srv := &Server{
		cfg: cfg,
		p:   user.NewProvisioners(),
		db:  db,
	}

	grpcServer := grpc.NewServer()
	registerGRPCServers(grpcServer, srv)
	srv.s = grpcServer

	if err := srv.bootstrapBlockchain(); err != nil {
		panic(err)
	}

	return srv, nil
}

func (s *Server) bootstrapBlockchain() error {
	log.Infoln("bootstrapping blockchain")

	provisioners, errFetch := s.db.FetchProvisioners()
	if errFetch != nil {
		provisioners = user.NewProvisioners()

		// Could not restore provisioners from DB.
		// Then we should regenerate DB from Genesis block.

		// Reconstruct Genesis Provisioners
		g := config.DecodeGenesis()
		if err := chain.ReconstructCommittee(provisioners, g); err != nil {
			return fmt.Errorf("couldn't reconstruct genesis committee: %v", err)
		}

		if err := s.db.PersistStakeContractAndHeight(provisioners, 0); err != nil {
			return fmt.Errorf("couldn't persist genesis committee: %v", err)
		}
	}

	h, err := s.db.FetchHeight()
	if err != nil {
		return fmt.Errorf("couldn't fetch height: %v", err)
	}

	log.WithField("height", h).Info("network state initialized")

	s.height = h
	s.p = provisioners

	return nil
}

func registerGRPCServers(grpcServer *grpc.Server, srv *Server) {
	log.Debugln("registering GRPC services")
	rusk.RegisterStateServer(grpcServer, srv)
	rusk.RegisterKeysServer(grpcServer, srv)
	rusk.RegisterBlindBidServiceServer(grpcServer, srv)
	rusk.RegisterBidServiceServer(grpcServer, srv)
	rusk.RegisterTransferServer(grpcServer, srv)
	rusk.RegisterStakeServiceServer(grpcServer, srv)
	rusk.RegisterWalletServer(grpcServer, srv)
	log.Debugln("GRPC services registered")
}

// Serve will start listening on a hardcoded IP and port. The server will then accept
// incoming gRPC requests.
func (s *Server) Serve(network, url string) error {
	log.WithField("addr", url).WithField("net", network).Infoln("starting GRPC server")

	if network == "unix" {
		// Remove obsolete unix socket file
		_ = os.Remove(url)
	}

	l, err := net.Listen(network, url)
	if err != nil {
		return err
	}

	go func() {
		if err := s.s.Serve(l); err != nil {
			logrus.WithError(err).Errorln("rusk mock server encountered an error")
		}
	}()

	log.Infoln("GRPC server started")
	return nil
}

// VerifyStateTransition simulates a state transition validation. The outcome is dictated
// by the server configuration.
func (s *Server) VerifyStateTransition(ctx context.Context, req *rusk.VerifyStateTransitionRequest) (*rusk.VerifyStateTransitionResponse, error) {
	log.Infoln("call received to VerifyStateTransition")
	defer log.Infoln("finished call to VerifyStateTransition")

	time.Sleep(stateTransitionDelay)

	if !s.cfg.PassStateTransitionValidation {
		indices := make([]uint64, len(req.Txs))
		for i := range indices {
			indices[i] = uint64(i)
		}

		return &rusk.VerifyStateTransitionResponse{
			FailedCalls: indices,
		}, nil
	}

	return &rusk.VerifyStateTransitionResponse{
		FailedCalls: make([]uint64, 0),
	}, nil
}

// ExecuteStateTransition simulates a state transition. The outcome is dictated by the server
// configuration.
func (s *Server) ExecuteStateTransition(ctx context.Context, req *rusk.ExecuteStateTransitionRequest) (*rusk.ExecuteStateTransitionResponse, error) {
	log.WithField("height", s.height).Infoln("call received to ExecuteStateTransition")
	defer log.Infoln("finished call to ExecuteStateTransition")

	time.Sleep(stateTransitionDelay)

	if err := s.addConsensusNodes(req.Txs, req.Height); err != nil {
		log.WithError(err).Errorln("could not add consensus nodes")
		return nil, err
	}

	return &rusk.ExecuteStateTransitionResponse{
		Success: s.cfg.PassStateTransition,
	}, nil
}

// GetProvisioners returns the current set of provisioners.
func (s *Server) GetProvisioners(ctx context.Context, req *rusk.GetProvisionersRequest) (*rusk.GetProvisionersResponse, error) {
	log.Infoln("call received to GetProvisioners")
	defer log.Infoln("finished call to GetProvisioners")

	provisioners, err := s.db.FetchProvisioners()
	if err == nil {
		s.p = provisioners
	}

	return &rusk.GetProvisionersResponse{
		Provisioners: legacy.ProvisionersToRuskCommittee(s.p),
	}, nil
}

// GetHeight returns height of the last call to ExecuteStateTransition.
func (s *Server) GetHeight(ctx context.Context, req *rusk.GetHeightRequest) (*rusk.GetHeightResponse, error) {
	log.Infoln("call received to GetHeight")
	defer log.Infoln("finished call to GetHeight")

	h, err := s.db.FetchHeight()
	if err != nil {
		return &rusk.GetHeightResponse{Height: 0}, err
	}

	return &rusk.GetHeightResponse{Height: h}, nil
}

func (s *Server) addConsensusNodes(txs []*rusk.Transaction, startHeight uint64) error {
	log.Debugln("adding consensus nodes")

	p := s.p.Copy()

	var updated bool

	for _, tx := range txs {
		if tx.Type == uint32(transactions.Stake) {
			payload := transactions.NewTransactionPayload()
			if err := transactions.UnmarshalTransactionPayload(bytes.NewBuffer(tx.Payload), payload); err != nil {
				return err
			}

			lock := binary.LittleEndian.Uint64(payload.CallData[0:8])

			var pk []byte
			if err := encoding.ReadVarBytes(bytes.NewBuffer(payload.CallData[8:]), &pk); err != nil {
				return err
			}

			value := binary.LittleEndian.Uint64(payload.SpendingProof[0:8])

			// Add grace period for stakes.
			stakeStartHeight := startHeight + stakeGracePeriod
			if err := p.Add(pk, value, stakeStartHeight, startHeight+lock-2); err != nil {
				return err
			}

			updated = true

			log.WithFields(logrus.Fields{
				"BLS key":      util.StringifyBytes(pk),
				"amount":       value,
				"start height": stakeStartHeight,
				"end height":   startHeight + lock - 2,
			}).Infoln("added provisioner")
		}
	}

	if updated {
		log.WithField("size", p.Set.Len()).Infoln("update provisioners db")
		// New provisioners added on last block.
		// Update ondisk copy of provisioners.
		if err := s.db.PersistStakeContractAndHeight(&p, startHeight); err != nil {
			return err
		}

		s.p = &p
	} else {
		// No new provisioners added, update height field only
		if err := s.db.PersistHeight(startHeight); err != nil {
			return err
		}
	}

	s.height = startHeight

	return nil
}

// GenerateScore returns a mocked Score.
// We do this entirely randomly, as score verification is completely up to
// the server configuration. This makes it easier for us to test different
// scenarios, and it greatly simplifies the bootstrapping of a network.
func (s *Server) GenerateScore(ctx context.Context, req *rusk.GenerateScoreRequest) (*rusk.GenerateScoreResponse, error) {
	log.Infoln("call received to GenerateScore")
	defer log.Infoln("finished call to GenerateScore")

	return nil, nil
}

// VerifyScore will return either true or false, depending on the server configuration.
func (s *Server) VerifyScore(ctx context.Context, req *rusk.VerifyScoreRequest) (*rusk.VerifyScoreResponse, error) {
	log.Infoln("call received to VerifyScore")
	defer log.Infoln("finished call to VerifyScore")

	return nil, nil
}

// GenerateKeys returns the server's wallet private key, and a stealth address.
// The response will contain Ristretto points under the hood.
func (s *Server) GenerateKeys(ctx context.Context, req *rusk.GenerateKeysRequest) (*rusk.GenerateKeysResponse, error) {
	log.Infoln("call received to GenerateKeys")
	defer log.Infoln("finished call to GenerateKeys")

	pSpend, err := crypto.RandEntropy(32)
	if err != nil {
		return nil, err
	}

	pk, err := crypto.RandEntropy(32)
	if err != nil {
		return nil, err
	}

	return &rusk.GenerateKeysResponse{
		Sk: &rusk.SecretKey{
			A: pSpend,
			B: make([]byte, 32),
		},
		Vk: &rusk.ViewKey{
			A:  make([]byte, 32),
			BG: make([]byte, 32),
		},
		Pk: &rusk.PublicKey{
			AG: pk,
			BG: make([]byte, 32),
		},
	}, nil
}

// GenerateStealthAddress returns a stealth address generated from the server's
// wallet public key.
func (s *Server) GenerateStealthAddress(ctx context.Context, req *rusk.PublicKey) (*rusk.StealthAddress, error) {
	log.Infoln("call received to GenerateStealthAddress")
	defer log.Infoln("finished call to GenerateStealthAddress")

	cpy := make([]byte, len(req.AG))
	copy(cpy, req.AG)

	return &rusk.StealthAddress{
		RG:  cpy,
		PkR: make([]byte, 0),
	}, nil
}

// NewTransfer creates a transaction and returns it to the caller.
func (s *Server) NewTransfer(ctx context.Context, req *rusk.TransferTransactionRequest) (*rusk.Transaction, error) {
	log.Infoln("call received to NewTransfer")
	defer log.Infoln("finished call to NewTransfer")

	anchor, err := crypto.RandEntropy(32)
	if err != nil {
		return nil, err
	}

	payload := &transactions.TransactionPayload{
		Anchor:        anchor,
		Nullifiers:    make([][]byte, 0),
		Notes:         make([]*transactions.Note, 0),
		Crossover:     transactions.MockCrossover(false),
		Fee:           transactions.MockFee(false),
		SpendingProof: make([]byte, 0),
		CallData:      make([]byte, 0),
	}

	buf := new(bytes.Buffer)
	if err := transactions.MarshalTransactionPayload(buf, payload); err != nil {
		return nil, err
	}

	// NOTE: None of this is gonna be checked so it doesn't matter what's in here.
	return &rusk.Transaction{
		Version: 0,
		Type:    0,
		Payload: buf.Bytes(),
	}, nil
}

// NewStake creates a staking transaction and returns it to the caller.
func (s *Server) NewStake(ctx context.Context, req *rusk.StakeTransactionRequest) (*rusk.Transaction, error) {
	log.Infoln("call received to NewStake")
	defer log.Infoln("finished call to NewStake")

	PubKeyBLS := make([]byte, len(req.PublicKeyBls))
	copy(PubKeyBLS, req.PublicKeyBls)

	calldata := new(bytes.Buffer)
	if err := encoding.WriteUint64LE(calldata, 250000); err != nil {
		return nil, err
	}

	if err := encoding.WriteVarBytes(calldata, PubKeyBLS[0:96]); err != nil {
		return nil, err
	}

	value := make([]byte, 8)
	binary.LittleEndian.PutUint64(value, req.Value)

	anchor, err := crypto.RandEntropy(32)
	if err != nil {
		return nil, err
	}

	payload := &transactions.TransactionPayload{
		Anchor:        anchor,
		Nullifiers:    make([][]byte, 0),
		Notes:         make([]*transactions.Note, 0),
		Crossover:     transactions.MockCrossover(false),
		Fee:           transactions.MockFee(false),
		SpendingProof: value,
		CallData:      calldata.Bytes(),
	}

	buf := new(bytes.Buffer)
	if err := transactions.MarshalTransactionPayload(buf, payload); err != nil {
		return nil, err
	}

	return &rusk.Transaction{
		Version: 0,
		Type:    4,
		Payload: buf.Bytes(),
	}, nil
}

// NewBid creates a bidding transaction and returns it to the caller.
func (s *Server) NewBid(ctx context.Context, req *rusk.BidTransactionRequest) (*rusk.BidTransaction, error) {
	log.Infoln("call received to NewBid")
	defer log.Infoln("finished call to NewBid")

	return nil, nil
}

// FindBid will return all of the bids for a given stealth address.
// TODO: implement.
func (s *Server) FindBid(ctx context.Context, req *rusk.FindBidRequest) (*rusk.BidList, error) {
	log.Infoln("call received to FindBid")
	defer log.Infoln("finished call to FindBid")

	return nil, nil
}

// FindStake will return a stake for a given public key.
// TODO: Implement.
func (s *Server) FindStake(ctx context.Context, req *rusk.FindStakeRequest) (*rusk.FindStakeResponse, error) {
	log.Infoln("call received to FindStake")
	defer log.Infoln("finished call to FindStake")

	return nil, nil
}

// GetBalance locked and unlocked balance values per a ViewKey.
func (s *Server) GetBalance(ctx context.Context, req *rusk.GetBalanceRequest) (*rusk.GetWalletBalanceResponse, error) {
	log.Infoln("call received to GetBalance")
	defer log.Infoln("finished call to GetBalance")

	resp := new(rusk.GetWalletBalanceResponse)

	resp.LockedBalance = 0
	resp.UnlockedBalance = 0
	return resp, nil
}

// Stop the rusk mock server.
func (s *Server) Stop() error {
	log.Infoln("stopping RUSK mock server")

	s.s.Stop()
	return nil
}
